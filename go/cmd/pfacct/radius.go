package main

import (
	"context"
	"database/sql"
	"encoding/binary"
	"errors"
	"fmt"
	"hash"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/OneOfOne/xxhash"
	"github.com/VividCortex/mysqlerr"
	cache "github.com/fdurand/go-cache"
	"github.com/go-sql-driver/mysql"

	"github.com/inverse-inc/go-radius"
	"github.com/inverse-inc/go-radius/dictionary"
	"github.com/inverse-inc/go-radius/inversedict"
	"github.com/inverse-inc/go-radius/rfc2865"
	"github.com/inverse-inc/go-radius/rfc2866"
	"github.com/inverse-inc/go-radius/rfc2869"
	"github.com/inverse-inc/go-utils/log"
	"github.com/inverse-inc/go-utils/mac"
	"github.com/inverse-inc/go-utils/sharedutils"
	"github.com/inverse-inc/packetfence/go/db"
	"github.com/inverse-inc/packetfence/go/pfconfigdriver"
)

const TRIGGER_TYPE_ACCOUNTING = "accounting"
const ACCOUNTING_POLICY_BANDWIDTH = "BandwidthExpired"
const ACCOUNTING_POLICY_TIME = "TimeExpired"

var radiusDictionary *dictionary.Dictionary

func (h *PfAcct) AddProxyState(packet *radius.Packet, r *radius.Request) *radius.Packet {
	state, err := rfc2865.ProxyState_Lookup(r.Packet)
	if err == nil {
		rfc2865.ProxyState_Add(packet, state)
	}
	return packet
}

func djb2Hash(s []byte) uint64 {
	hash := uint64(5381)
	for _, c := range s {
		hash = ((hash << 5) + hash) + uint64(c)
		// the above line is an optimized version of the following line:
		//hash = hash * 33 + uint64(c)
		// which is easier to read and understand...
	}

	return hash
}

func (h *PfAcct) ServeRADIUS(w radius.ResponseWriter, r *radius.Request) {
	switch r.Code {
	case radius.CodeAccountingRequest:
		h.HandleAccounting(w, r)
	case radius.CodeStatusServer:
		h.HandleStatusServer(w, r)
	}
}

func (h *PfAcct) hasher() hash.Hash64 {
	return xxhash.New64()
}

func (h *PfAcct) HandleStatusServer(w radius.ResponseWriter, r *radius.Request) {
	w.Write(h.AddProxyState(r.Response(radius.CodeAccessAccept), r))
}

func (h *PfAcct) HandleAccounting(w radius.ResponseWriter, r *radius.Request) {
	ctx := r.Context()
	defer h.NewTiming().Send("pfacct.HandleAccountingRequest")
	status := rfc2866.AcctStatusType_Get(r.Packet)
	if status > rfc2866.AcctStatusType_Value_InterimUpdate {
		outPacket := r.Response(radius.CodeAccountingResponse)
		rfc2865.ReplyMessage_SetString(outPacket, "Accounting OK")
		logInfo(ctx, fmt.Sprintf("Accounting status of %s ignored", status.String()))
		w.Write(h.AddProxyState(outPacket, r))
		return
	}

	iSwitchInfo := ctx.Value(switchInfoKey)
	if iSwitchInfo == nil {
		panic("SwitchInfo: not found")
	}

	switchInfo := iSwitchInfo.(*SwitchInfo)
	callingStation := rfc2865.CallingStationID_GetString(r.Packet)
	mac, _ := mac.NewFromString(callingStation)
	rr := radiusRequest{
		r:          r,
		status:     status,
		switchInfo: switchInfo,
		mac:        mac,
	}

	h.sendRadiusRequestToQueue(rr)
	outPacket := r.Response(radius.CodeAccountingResponse)
	rfc2865.ReplyMessage_SetString(outPacket, "Accounting OK")
	w.Write(h.AddProxyState(outPacket, r))
	// h.handleAccountingRequest(w, r, switchInfo, mac)
	// h.Dispatcher.SubmitJob(Work(func() { h.handleAccountingRequest(r, switchInfo) }))
}

func (h *PfAcct) sendRadiusRequestToQueue(rr radiusRequest) {
	queueIndex := djb2Hash(rr.mac[:]) % uint64(len(h.radiusRequests))
	h.radiusRequests[queueIndex] <- rr
}

func (h *PfAcct) handleAccountingRequest(rr radiusRequest) {
	r, switchInfo, mac, status := rr.r, rr.switchInfo, rr.mac, rr.status
	defer h.NewTiming().Send("pfacct.accounting." + rr.status.String())
	ctx := r.Context()
	in_bytes := int64(rfc2866.AcctInputOctets_Get(r.Packet))
	out_bytes := int64(rfc2866.AcctOutputOctets_Get(r.Packet))
	giga_in_bytes := int64(rfc2869.AcctInputGigawords_Get(r.Packet))
	giga_out_bytes := int64(rfc2869.AcctOutputGigawords_Get(r.Packet))
	out_bytes += giga_out_bytes << 32
	in_bytes += giga_in_bytes << 32
	unique_session_id := h.accountingUniqueSessionId(r)
	err := h.updateNodeOnlineOfflineOnline(status, mac, unique_session_id)
	if err != nil {
		logError(ctx, fmt.Sprintf("Error updating online offline status: %s", err.Error()))
	}

	timestamp, err := rfc2869.EventTimestamp_Lookup(r.Packet)
	if err != nil {
		timestamp = time.Now()
	} else {
		if timestamp.Before(time.Now().AddDate(0, 0, -1)) {
			logError(
				ctx,
				fmt.Sprintf(
					"[mac:%s, id:%d, timestamp:%s] skip packet timestamp is older than a day",
					mac.String(),
					unique_session_id,
					timestamp.String(),
				),
			)
			return
		}
	}

	timestamp = timestamp.Truncate(h.TimeDuration)
	node_id := mac.NodeId(uint16(switchInfo.TenantId))
	if h.ProcessBandwidthAcct {
		if err := h.InsertBandwidthAccounting(
			status,
			node_id,
			mac.String(),
			unique_session_id,
			timestamp,
			in_bytes,
			out_bytes,
		); err != nil {
			logError(ctx, "InsertBandwidthAccounting: "+err.Error())
		}

		if status == rfc2866.AcctStatusType_Value_Stop {
			h.CloseSession(node_id, unique_session_id)
		}
	}

	h.sendRadiusAccounting(r, switchInfo)
	h.handleTimeBalance(r, switchInfo, unique_session_id)
	h.handleBandwidthBalance(r, switchInfo, in_bytes+out_bytes)
}

func (h *PfAcct) handleTimeBalance(r *radius.Request, switchInfo *SwitchInfo, unique_session uint64) {
	timebalance := int64(rfc2866.AcctSessionTime_Get(r.Packet))
	if timebalance == 0 {
		return
	}
	ctx := r.Context()

	callingStation := rfc2865.CallingStationID_GetString(r.Packet)
	mac, _ := mac.NewFromString(callingStation)
	status := rfc2866.AcctStatusType_Get(r.Packet)
	isUnreg, _ := h.IsUnreg(mac.String())
	ns := h.getNodeSessionFromCache(unique_session)
	if status == rfc2866.AcctStatusType_Value_Stop {
		defer h.deleteNodeSessionFromCache(unique_session)
		if isUnreg {
			return
		}

		if ns != nil {
			timebalance -= ns.timeBalance
			if timebalance < 0 {
				timebalance = 0
			}
		}

		ok, err := h.NodeTimeBalanceSubtract(mac, timebalance)
		if err != nil {
			logError(ctx, "NodeTimeBalanceSubtract: "+err.Error())
			return
		}

		if ok {
			if ok, err = h.IsNodeTimeBalanceZero(mac); ok {
				if err := h.AAAClient.Notify(ctx, "trigger_security_event", []interface{}{"type", TRIGGER_TYPE_ACCOUNTING, "mac", mac.String(), "tid", ACCOUNTING_POLICY_TIME}); err != nil {
					logError(ctx, "IsNodeTimeBalanceZero: "+err.Error())
				}
			}
		}
	} else {
		if isUnreg {
			if ns == nil {
				h.setNodeSessionCache(unique_session, &nodeSession{timeBalance: -1})
			} else {
				ns.timeBalance = -1
			}
			return
		}

		if ns != nil {
			if ns.timeBalance == -1 {
				ns.timeBalance = timebalance
			} else {
				timebalance -= ns.timeBalance
				if timebalance < 0 {
					timebalance = 0
				}
			}
		}

		if timebalance > 0 {
			ok, err := h.SoftNodeTimeBalanceUpdate(mac, timebalance)
			if err != nil {
				logError(ctx, "SoftNodeTimeBalanceUpdate: "+err.Error())
				return
			}
			if ok {
				if err := h.AAAClient.Notify(ctx, "trigger_security_event", []interface{}{"type", TRIGGER_TYPE_ACCOUNTING, "mac", mac.String(), "tid", ACCOUNTING_POLICY_TIME}); err != nil {
					logError(ctx, "Notify trigger_security_event: "+err.Error())
				}
			}
		}
	}
}

func (h *PfAcct) handleBandwidthBalance(r *radius.Request, switchInfo *SwitchInfo, balance int64) {
	if balance == 0 {
		return
	}

	ctx := r.Context()
	callingStation := rfc2865.CallingStationID_GetString(r.Packet)
	mac, _ := mac.NewFromString(callingStation)
	status := rfc2866.AcctStatusType_Get(r.Packet)
	if status == rfc2866.AcctStatusType_Value_Stop {
		ok, err := h.NodeBandwidthBalanceSubtract(mac, balance)
		if err != nil {
			logError(ctx, "NodeBandwidthBalanceSubtract: "+err.Error())
			return
		}
		if ok {
			if ok, err = h.IsNodeBandwidthBalanceZero(mac); ok {
				if err := h.AAAClient.Notify(ctx, "trigger_security_event", []interface{}{"type", TRIGGER_TYPE_ACCOUNTING, "mac", mac.String(), "tid", ACCOUNTING_POLICY_BANDWIDTH}); err != nil {
					logError(ctx, "IsNodeBandwidthBalanceZero: "+err.Error())
				}
			}
		}
	} else {
		ok, err := h.SoftNodeBandwidthBalanceUpdate(mac, balance)
		if err != nil {
			logError(ctx, "SoftNodeBandwidthBalanceUpdate: "+err.Error())
			return
		}
		if ok {
			if err := h.AAAClient.Notify(ctx, "trigger_security_event", []interface{}{"type", TRIGGER_TYPE_ACCOUNTING, "mac", mac.String(), "tid", ACCOUNTING_POLICY_BANDWIDTH}); err != nil {
				logError(ctx, "Notify trigger_security_event: "+err.Error())
			}
		}
	}
}

func (h *PfAcct) accountingUniqueSessionId(r *radius.Request) uint64 {
	username := rfc2865.UserName_Get(r.Packet)
	callingStation := rfc2865.CallingStationID_Get(r.Packet)
	acctSessionId := rfc2866.AcctSessionID_Get(r.Packet)
	hash := h.hasher()
	hash.Write(username)
	hash.Write([]byte{','})
	hash.Write(callingStation)
	hash.Write([]byte{','})
	hash.Write(acctSessionId)
	return hash.Sum64()
}

func (h *PfAcct) sendRadiusAccounting(r *radius.Request, switchInfo *SwitchInfo) {
	ctx := r.Context()
	attr := packetToMap(ctx, r.Packet)
	attr["PF_HEADERS"] = map[string]string{
		"X-FreeRADIUS-Server":  "packetfence",
		"X-FreeRADIUS-Section": "accounting",
	}

	if val, ok := attr["NAS-IP-Address"]; !ok || val == "0.0.0.0" {
		attr["NAS-IP-Address"] = strings.Split(r.RemoteAddr.String(), ":")[0]
		logWarn(ctx, fmt.Sprintf("Empty NAS-IP-Address, using the source IP address of the packet (%s)", attr["NAS-IP-Address"]))
	}

	if _, err := h.AAAClient.Call(ctx, "radius_accounting", attr); err != nil {
		logError(ctx, err.Error())
	}
}

func (h *PfAcct) radiusListen(w *sync.WaitGroup) *radius.PacketServer {

	var ctx = context.Background()

	var RADIUSinterfaces pfconfigdriver.RADIUSInts
	pfconfigdriver.FetchDecodeSocket(ctx, &RADIUSinterfaces)

	ipRADIUS := []string{"0.0.0.0"}

	var intRADIUS []*net.UDPConn

	for _, adresse := range sharedutils.RemoveDuplicates(ipRADIUS) {

		port := "1813"

		addr, err := net.ResolveUDPAddr("udp4", adresse+":"+port)
		if err != nil {
			panic(err)
		}
		pc, err := net.ListenUDP("udp4", addr)
		if err != nil {
			panic(err)
		}
		intRADIUS = append(intRADIUS, pc)
	}

	server := &radius.PacketServer{
		Handler:      h,
		SecretSource: h,
	}

	for _, pc := range intRADIUS {
		w.Add(1)
		go func(pc *net.UDPConn) {
			if err := server.Serve(pc); err != radius.ErrServerShutdown {
				panic(err)
			}

			w.Done()
		}(pc)
	}

	return server
}

type contextKey int

const (
	switchInfoKey contextKey = iota
)

func (h *PfAcct) RADIUSSecret(ctx context.Context, remoteAddr net.Addr, raw []byte) ([]byte, context.Context, error) {
	srcIpAddr := remoteAddr.(*net.UDPAddr).IP.String()
	var nasIpAddr string
	var err error
	var macStr string
	err = checkPacket(raw)
	if err != nil {
		logError(h.LoggerCtx, "RADIUSSecret: "+err.Error())
		return nil, nil, err
	}

	attrs, err := radius.ParseAttributes(raw[20:])
	if err != nil {
		logError(h.LoggerCtx, "RADIUSSecret: "+err.Error())
		return nil, nil, err
	}

	attr, ok := attrs.Lookup(rfc2865.CalledStationID_Type)
	if !ok {
		macStr = ""
	} else {
		mac, err := mac.NewFromString(string(attr))
		if err != nil {
			macStr = ""
		} else {
			macStr = mac.String()
		}
	}

	if attr, ok = attrs.Lookup(rfc2865.NASIPAddress_Type); ok {
		if val, err := radius.IPAddr(attr); err == nil {
			nasIpAddr = val.String()
		}
	}

	switchInfo, err := h.SwitchLookup(macStr, srcIpAddr, nasIpAddr)
	if err != nil {
		logError(h.LoggerCtx, "RADIUSSecret: Switch '"+srcIpAddr+"' not found :"+err.Error())
		return nil, nil, err
	}

	packet, err := radius.Parse(raw, []byte(switchInfo.Secret))
	if err != nil {
		logError(h.LoggerCtx, "RADIUSSecret: "+err.Error())
		return nil, nil, err
	}

	// If the request overrides the tenant ID, we create a copy of the switchInfo and return it with an updated tenant ID
	if val := inversedict.PacketFenceTenantID_Get(packet); val != 0 {
		switchInfo2 := *switchInfo
		switchInfo2.TenantId = int(val)
		return []byte(switchInfo.Secret), log.TranferLogContext(h.LoggerCtx, context.WithValue(ctx, switchInfoKey, &switchInfo2)), nil
	} else {
		return []byte(switchInfo.Secret), log.TranferLogContext(h.LoggerCtx, context.WithValue(ctx, switchInfoKey, switchInfo)), nil
	}

}

type Error string

func (e Error) Error() string { return string(e) }

const PacketTooSmall = Error("radius: packet not at least 20 bytes long")
const PacketInvalidLength = Error("radius: invalid packet length")

func checkPacket(raw []byte) error {
	if len(raw) < 20 {
		return PacketTooSmall
	}

	length := int(binary.BigEndian.Uint16(raw[2:4]))
	if length < 20 || length > radius.MaxPacketLength || len(raw) != length {
		return PacketInvalidLength
	}

	return nil
}

func packetToMap(ctx context.Context, p *radius.Packet) map[string]interface{} {

	attributes := make(map[string]interface{})
	for i, attr := range p.Attributes {
		if rfc2865.VendorSpecific_Type == i {
			for _, vattrs := range attr {
				id, vsa, err := radius.VendorSpecific(vattrs)
				if err != nil {
					log.LoggerWContext(ctx).Error(fmt.Sprintf("Unknown vendor id: %d", id))
					continue
				}

				v := radiusDictionary.GetVendorByNumber(uint(id))
				if v == nil {
					log.LoggerWContext(ctx).Error(fmt.Sprintf("Unknown vendor id: %d", id))
					continue
				}

				for len(vsa) >= 3 {
					vsaTyp, vsaLen := vsa[0], vsa[1]
					data := vsa[2:int(vsaLen)]
					a := dictionary.AttributeByOID(v.Attributes, []int{int(vsaTyp)})
					vsa = vsa[int(vsaLen):]
					if a == nil {
						continue
					}

					addAttributeToMap(ctx, attributes, a, radius.Attribute(data))
				}
			}
		} else {
			a := radiusDictionary.GetAttributeByOID([]int{int(i)})
			if a == nil {
				log.LoggerWContext(ctx).Error(fmt.Sprintf("Unknown Attribute: %d", int(i)))
				continue
			}

			addAttributeToMap(ctx, attributes, a, attr[0])
		}
	}

	if val, found := attributes["Calling-Station-Id"]; found {
		if mac, err := mac.NewFromString(val.(string)); err == nil {
			attributes["Calling-Station-Id"] = mac.String()
		}
	}

	return attributes
}

func addAttributeToMap(ctx context.Context, attributes map[string]interface{}, da *dictionary.Attribute, attr radius.Attribute) {
	var item interface{} = nil
	switch da.Type {
	case dictionary.AttributeString:
		item = radius.String(attr)
	case dictionary.AttributeInteger:
		i, err := radius.Integer(attr)
		if err == nil {
			item = i
		}
	case dictionary.AttributeInteger64:
		i, err := radius.Integer64(attr)
		if err == nil {
			item = i
		}
	case dictionary.AttributeIPAddr:
		i, err := radius.IPAddr(attr)
		if err == nil {
			item = i.String()
		}
	case dictionary.AttributeDate:
		i, err := radius.Date(attr)
		if err == nil {
			item = i.String()
		}
	}

	if item != nil {
		if old, found := attributes[da.Name]; found {
			switch old.(type) {
			case []interface{}:
				attributes[da.Name] = append(old.([]interface{}), item)
			default:
				attributes[da.Name] = []interface{}{old, item}
			}
		} else {
			attributes[da.Name] = item
		}
	} else {
		logDebug(ctx, fmt.Sprintf("Serialization of data type %s for %s not handled\n", da.Type, da.Name))
	}
}

func logError(ctx context.Context, msg string) {
	log.LoggerWContext(ctx).Error(msg)
}

func logWarn(ctx context.Context, msg string) {
	log.LoggerWContext(ctx).Warn(msg)
}

func logInfo(ctx context.Context, msg string) {
	log.LoggerWContext(ctx).Info(msg)
}

func logDebug(ctx context.Context, msg string) {
	log.LoggerWContext(ctx).Debug(msg)
}

type RadiusStatements struct {
	switchLookup                    *sql.Stmt
	insertBandwidthAccountingStart  *sql.Stmt
	insertBandwidthAccountingUpdate *sql.Stmt
	softNodeTimeBalanceUpdate       *sql.Stmt
	nodeTimeBalanceSubtract         *sql.Stmt
	nodeTimeBalance                 *sql.Stmt
	isNodeTimeBalanceZero           *sql.Stmt
	softNodeBandwidthBalanceUpdate  *sql.Stmt
	nodeBandwidthBalanceSubtract    *sql.Stmt
	nodeBandwidthBalance            *sql.Stmt
	isNodeBandwidthBalanceZero      *sql.Stmt
	isUnreg                         *sql.Stmt
	closeSession                    *sql.Stmt
	nodeOnlineOffLineStartUpdate    *sql.Stmt
	nodeOnlineOffLineStop           *sql.Stmt
}

func setupStmt(db *sql.DB, stmt **sql.Stmt, sql string) {
	var err error
	if *stmt, err = db.Prepare(sql); err != nil {
		log.Logger().Error(fmt.Sprintf("Failed to prepare statement '%s': %s", sql, err))
		go func() {
			var err error
			for {
				time.Sleep(5 * time.Second)
				if *stmt, err = db.Prepare(sql); err == nil {
					break
				}

				log.Logger().Error(fmt.Sprintf("Failed to prepare statement '%s': %s", sql, err))
			}
		}()
	}
}

func isErrorRetryable(err error) bool {
	driverErr, ok := err.(*mysql.MySQLError)
	if !ok {
		return false
	}

	return driverErr.Number == mysqlerr.ER_LOCK_DEADLOCK || driverErr.Number == mysqlerr.ER_LOCK_WAIT_TIMEOUT
}

func tryExecute(retry int, pause time.Duration, stmt *sql.Stmt, args ...interface{}) (sql.Result, error) {
	var err error
	var res sql.Result
	for retry >= 0 {
		retry--
		res, err = stmt.Exec(args...)
		if err == nil {
			break
		}

		if isErrorRetryable(err) {
			if pause != 0 {
				time.Sleep(pause)
			}
			continue
		}

		break
	}

	return res, err
}

func (rs *RadiusStatements) Setup(db *sql.DB) {
	setupStmt(db, &rs.switchLookup, `
		SELECT nasname, secret, unique_session_attributes
		FROM (
			SELECT nasname, secret, unique_session_attributes, 0 as o FROM radius_nas WHERE nasname = ?
			UNION ALL
			( SELECT nasname, secret, unique_session_attributes, 1 as o from radius_nas WHERE INET_ATON(?) BETWEEN start_ip AND end_ip order by range_length limit 1)
			UNION ALL
			( SELECT nasname, secret, unique_session_attributes , 2 as o from radius_nas WHERE INET_ATON(?) BETWEEN start_ip AND end_ip order by range_length limit 1)

		) as x ORDER BY o LIMIT 1;
	`)

	setupStmt(db, &rs.insertBandwidthAccountingStart, `
        INSERT INTO bandwidth_accounting (node_id, mac, unique_session_id, time_bucket, in_bytes, out_bytes, source_type)
            SELECT ? as node_id, ? AS mac, ? AS unique_session_id, ? AS time_bucket, in_bytes, out_bytes, "radius" FROM (
                SELECT GREATEST(? - IFNULL(SUM(in_bytes), 0), 0) AS in_bytes, GREATEST(? - IFNULL(SUM(out_bytes), 0), 0) AS out_bytes FROM bandwidth_accounting WHERE node_id = ? AND unique_session_id = ? AND time_bucket != ?
            ) AS y
        ON DUPLICATE KEY UPDATE in_bytes = VALUES(in_bytes), out_bytes = VALUES(out_bytes), last_updated = NOW();
	`)

	setupStmt(db, &rs.insertBandwidthAccountingUpdate, `
        INSERT INTO bandwidth_accounting (node_id, mac, unique_session_id, time_bucket, in_bytes, out_bytes, source_type)
            SELECT ? as node_id, ? AS mac, ? AS unique_session_id, ? AS time_bucket, in_bytes, out_bytes, "radius" FROM (
                SELECT * FROM (
                    SELECT GREATEST(? - IFNULL(SUM(in_bytes), 0), 0) AS in_bytes, GREATEST(? - IFNULL(SUM(out_bytes), 0), 0) AS out_bytes, COUNT(1) AS entries FROM bandwidth_accounting WHERE node_id = ? AND unique_session_id = ? AND time_bucket != ?
                ) AS sum_bytes WHERE in_bytes !=0 OR out_bytes != 0 OR entries = 0
            ) AS y
        ON DUPLICATE KEY UPDATE in_bytes = VALUES(in_bytes), out_bytes = VALUES(out_bytes), last_updated = NOW();
	`)

	setupStmt(db, &rs.softNodeTimeBalanceUpdate, `
        UPDATE node set time_balance = 0 WHERE mac = ? AND time_balance <= ? AND (status = "reg" || DATE_SUB(NOW(), INTERVAL 5 MINUTE) > regdate);
	`)

	setupStmt(db, &rs.softNodeBandwidthBalanceUpdate, `
        UPDATE node set bandwidth_balance = 0 WHERE mac = ? AND bandwidth_balance <= ? AND (status = "reg" || DATE_SUB(NOW(), INTERVAL 5 MINUTE) > regdate );
	`)

	setupStmt(db, &rs.nodeTimeBalanceSubtract, `
        UPDATE node set time_balance = GREATEST(CAST(time_balance AS SIGNED) - ?, 0) WHERE mac = ? AND time_balance IS NOT NULL AND (status = "reg" || DATE_SUB(NOW(), INTERVAL 5 MINUTE) > regdate );
	`)

	setupStmt(db, &rs.isUnreg, `
        SELECT 1 FROM node WHERE mac = ? AND status = 'unreg'
	`)

	setupStmt(db, &rs.nodeBandwidthBalanceSubtract, `
        UPDATE node set bandwidth_balance = GREATEST(CAST(bandwidth_balance AS SIGNED) - ?, 0) WHERE mac = ? AND bandwidth_balance IS NOT NULL AND (status = "reg" || DATE_SUB(NOW(), INTERVAL 5 MINUTE) > regdate );
	`)

	setupStmt(db, &rs.nodeTimeBalance, `
        SELECT time_balance FROM node WHERE mac = ? AND time_balance IS NOT NULL;
	`)

	setupStmt(db, &rs.nodeBandwidthBalance, `
        SELECT bandwidth_balance FROM node WHERE mac = ? AND bandwidth_balance IS NOT NULL;
	`)

	setupStmt(db, &rs.isNodeTimeBalanceZero, `
        SELECT 1 FROM node WHERE mac = ? AND time_balance = 0;
	`)

	setupStmt(db, &rs.isNodeBandwidthBalanceZero, `
        SELECT 1 FROM node WHERE mac = ? AND bandwidth_balance = 0;
	`)

	setupStmt(db, &rs.closeSession, `
        UPDATE bandwidth_accounting SET last_updated = '0000-00-00 00:00:00' WHERE node_id = ? AND unique_session_id = ?;
	`)

	setupStmt(db, &rs.nodeOnlineOffLineStop, `
        UPDATE node_current_session SET is_online = 0 WHERE mac = ? AND last_session_id = ?;
       `)

	setupStmt(db, &rs.nodeOnlineOffLineStartUpdate, `
		INSERT INTO node_current_session (mac, last_session_id, updated, is_online) VALUES (?, ?, NOW(), 1)
        ON DUPLICATE KEY UPDATE updated = VALUES(updated), last_session_id = VALUES(last_session_id), is_online =1 ;
       `)

}

func (rs *RadiusStatements) updateNodeOnlineOfflineOnline(status rfc2866.AcctStatusType, mac mac.Mac, session_id uint64) error {
	var err error = nil
	switch status {
	default:
		err = errors.New("Invalid status")
	case rfc2866.AcctStatusType_Value_Start, rfc2866.AcctStatusType_Value_InterimUpdate:
		_, err = tryExecute(
			3,
			10*time.Millisecond,
			rs.nodeOnlineOffLineStartUpdate,
			mac.String(),
			session_id,
		)
	case rfc2866.AcctStatusType_Value_Stop:
		_, err = tryExecute(
			3,
			10*time.Millisecond,
			rs.nodeOnlineOffLineStop,
			mac.String(),
			session_id,
		)
	}

	return err
}

func (rs *RadiusStatements) IsUnreg(mac string) (bool, error) {
	found := 0
	err := rs.isUnreg.QueryRow(mac).Scan(&found)
	return found == 1, err
}

func (rs *RadiusStatements) CloseSession(node_id, unique_session_id uint64) (int64, error) {
	result, err := rs.closeSession.Exec(node_id, unique_session_id)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

func (rs *RadiusStatements) IsNodeTimeBalanceZero(mac mac.Mac) (bool, error) {
	found := 0
	err := rs.isNodeTimeBalanceZero.QueryRow(mac).Scan(&found)
	return found == 1, err
}

func (rs *RadiusStatements) SoftNodeTimeBalanceUpdate(mac mac.Mac, balance int64) (bool, error) {
	result, err := rs.softNodeTimeBalanceUpdate.Exec(mac.String(), balance)
	if err != nil {
		return false, err
	}

	if count, err := result.RowsAffected(); count <= 0 || err != nil {
		return false, err
	}

	return true, nil
}

func (rs *RadiusStatements) NodeTimeBalanceSubtract(mac mac.Mac, balance int64) (bool, error) {
	result, err := rs.nodeTimeBalanceSubtract.Exec(balance, mac.String())
	if err != nil {
		return false, err
	}

	if count, err := result.RowsAffected(); count <= 0 || err != nil {
		return false, err
	}

	return true, nil
}

func (rs *RadiusStatements) IsNodeBandwidthBalanceZero(mac mac.Mac) (bool, error) {
	found := 0
	err := rs.isNodeBandwidthBalanceZero.QueryRow(mac).Scan(&found)
	return found == 1, err
}

func (rs *RadiusStatements) SoftNodeBandwidthBalanceUpdate(mac mac.Mac, balance int64) (bool, error) {
	result, err := rs.softNodeBandwidthBalanceUpdate.Exec(mac.String(), balance)
	if err != nil {
		return false, err
	}

	if count, err := result.RowsAffected(); count <= 0 || err != nil {
		return false, err
	}

	return true, nil
}

func (rs *RadiusStatements) NodeBandwidthBalanceSubtract(mac mac.Mac, balance int64) (bool, error) {
	result, err := rs.nodeBandwidthBalanceSubtract.Exec(balance, mac.String())
	if err != nil {
		return false, err
	}

	if count, err := result.RowsAffected(); count <= 0 || err != nil {
		return false, err
	}

	return true, nil
}

type SwitchInfo struct {
	Nasname, Secret  string
	TenantId         int
	RadiusAttributes db.CsvArray
}

func (h *PfAcct) SwitchLookup(mac, srcIp, nasIp string) (*SwitchInfo, error) {
	key := mac + ":" + srcIp + ":" + nasIp
	if item, found := h.SwitchInfoCache.Get(key); found {
		return item.(*SwitchInfo), nil
	}

	switchInfo := &SwitchInfo{}
	err := h.switchLookup.QueryRow(mac, nasIp, srcIp).Scan(&switchInfo.Nasname, &switchInfo.Secret, &switchInfo.RadiusAttributes)
	if err != nil {
		return nil, err
	}

	if h.isProxied {
		switchInfo.Secret = h.localSecret
	}

	h.SwitchInfoCache.Set(key, switchInfo, cache.DefaultExpiration)
	return switchInfo, nil
}

func (h *PfAcct) updateTimeBalance(isUnreg bool, status rfc2866.AcctStatusType, timebalance int64, unique_session uint64) int64 {
	ns := h.getNodeSessionFromCache(unique_session)
	if ns == nil {
	}

	return timebalance
}

func (h *PfAcct) InsertBandwidthAccounting(status rfc2866.AcctStatusType, node_id uint64, mac string, unique_session uint64, bucket time.Time, in_bytes int64, out_bytes int64) error {
	var err error
	if status == rfc2866.AcctStatusType_Value_Start {
		h.SetAcctSession(node_id, unique_session, &AcctSession{in_bytes: in_bytes, out_bytes: out_bytes})
		_, err = tryExecute(
			3,
			time.Millisecond*10,
			h.insertBandwidthAccountingStart,
			node_id,
			mac,
			unique_session,
			bucket,
			in_bytes,
			out_bytes,
			node_id,
			unique_session,
			bucket,
		)
	} else {
		s := h.GetAcctSession(node_id, unique_session)
		if s != nil && s.in_bytes == in_bytes && s.out_bytes == out_bytes {
			return nil
		}

		h.SetAcctSession(node_id, unique_session, &AcctSession{in_bytes: in_bytes, out_bytes: out_bytes})
		_, err = tryExecute(
			3,
			time.Millisecond*10,
			h.insertBandwidthAccountingUpdate,
			node_id,
			mac,
			unique_session,
			bucket,
			in_bytes,
			out_bytes,
			node_id,
			unique_session,
			bucket,
		)
	}
	return err
}

func init() {
	parser := &dictionary.Parser{
		Opener: &dictionary.FileSystemOpener{
			Root: "/usr/share/freeradius",
		},
		IgnoreIdenticalAttributes:  true,
		IgnoreUnknownAttributeType: true,
	}

	var err error
	if radiusDictionary, err = parser.ParseFile("/usr/local/pf/raddb/dictionary.pfacct"); err != nil {
		panic(err)
	}

}
