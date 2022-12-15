import { BaseViewCollectionItem } from '../../_components/new/'
import {
  BaseFormButtonBar,
  BaseFormGroupChosenMultiple,
  BaseFormGroupChosenOne,
  BaseFormGroupFileUpload,
  BaseFormGroupInput,
  BaseFormGroupInputNumber,
  BaseFormGroupInputPassword,
  BaseFormGroupTextarea,
  BaseFormGroupToggle,
  BaseFormGroupToggleDisabledEnabled,
  BaseFormGroupToggleNoYes,
  BaseServices,
} from '@/components/new/'
import {
  BaseFormGroupIntervalUnit
} from '@/views/Configuration/_components/new/'
import BaseFormGroupActiveDirectoryPasswordTest from './BaseFormGroupActiveDirectoryPasswordTest'
import BaseFormGroupAdministrationRules from './BaseFormGroupAdministrationRules'
import BaseFormGroupAuthenticationRules from './BaseFormGroupAuthenticationRules'
import BaseFormGroupDomains from './BaseFormGroupDomains'
import BaseFormGroupHostPortEncryption from './BaseFormGroupHostPortEncryption'
import BaseFormGroupPersonMappings from './BaseFormGroupPersonMappings'
import BaseFormGroupProtocolHostPort from './BaseFormGroupProtocolHostPort'
import BaseFormGroupServerAddressPort from './BaseFormGroupServerAddressPort'
import ButtonSamlMetaData from './ButtonSamlMetaData'
import TheForm from './TheForm'
import TheView from './TheView'

export {
  BaseViewCollectionItem                    as BaseView,
  BaseFormButtonBar                         as FormButtonBar,

  BaseFormGroupInput                        as FormGroupAccessScope,
  BaseFormGroupInput                        as FormGroupAccessTokenParam,
  BaseFormGroupInput                        as FormGroupAccessTokenPath,
  BaseFormGroupInput                        as FormGroupAccountSid,
  BaseFormGroupInput                        as FormGroupActivationDomain,
  BaseFormGroupAdministrationRules          as FormGroupAdministrationRules,
  BaseFormGroupDomains                      as FormGroupAllowedDomains,
  BaseFormGroupToggleNoYes                  as FormGroupAllowLocaldomain,
  BaseFormGroupInput                        as FormGroupApiKey,
  BaseFormGroupInputPassword                as FormGroupApiPassword,
  BaseFormGroupInput                        as FormGroupApiUsername,
  BaseFormGroupInput                        as FormGroupApiLoginIdentifier,
  BaseFormGroupInput                        as FormGroupAuthenticateRealm,
  BaseFormGroupAuthenticationRules          as FormGroupAuthenticationRules,
  BaseFormGroupInput                        as FormGroupAuthenticationUrl,
  BaseFormGroupChosenOne                    as FormGroupAuthorizationSourceIdentifier,
  BaseFormGroupInput                        as FormGroupAuthorizationUrl,
  BaseFormGroupInput                        as FormGroupAuthorizePath,
  BaseFormGroupInputNumber                  as FormGroupAuthListeningPort,
  BaseFormGroupInput                        as FormGroupAuthToken,
  BaseFormGroupDomains                      as FormGroupBannedDomains,
  BaseFormGroupInput                        as FormGroupBaseDn,
  BaseFormGroupChosenOne                    as FormGroupBaseUrl,
  BaseFormGroupInput                        as FormGroupBindDn,
  BaseFormGroupFileUpload                   as FormGroupCaFile,
  BaseFormGroupInput                        as FormGroupCaPath,
  BaseFormGroupToggle                       as FormGroupCacheMatch,
  BaseFormGroupFileUpload                   as FormGroupCertFile,
  BaseFormGroupInput                        as FormGroupCertIdentifier,
  BaseFormGroupFileUpload                   as FormGroupClientCertFile,
  BaseFormGroupInput                        as FormGroupClientIdentifier,
  BaseFormGroupFileUpload                   as FormGroupClientKeyFile,
  BaseFormGroupInput                        as FormGroupClientSecret,
  BaseFormGroupInputNumber                  as FormGroupConnectionTimeout,
  BaseFormGroupToggleNoYes                  as FormGroupCreateLocalAccount,
  BaseFormGroupChosenOne                    as FormGroupCurrency,
  BaseFormGroupToggle                       as FormGroupCustomerPortal,
  BaseFormGroupInputNumber                  as FormGroupDeadDuration,
  BaseFormGroupInput                        as FormGroupDescription,
  BaseFormGroupInput                        as FormGroupDirectBaseUrl,
  BaseFormGroupInput                        as FormGroupDomains,
  BaseFormGroupIntervalUnit                 as FormGroupEmailActivationTimeout,
  BaseFormGroupInput                        as FormGroupEmailAddress,
  BaseFormGroupInput                        as FormGroupEmailAttribute,
  BaseFormGroupToggleNoYes                  as FormGroupEmailRequired,
  BaseFormGroupInput                        as FormGroupGroupHeader,
  BaseFormGroupChosenOne                    as FormGroupHashPasswords,
  BaseFormGroupInput                        as FormGroupHost,
  BaseFormGroupHostPortEncryption           as FormGroupHostPortEncryption,
  BaseFormGroupInput                        as FormGroupIdentifier,
  BaseFormGroupFileUpload                   as FormGroupIdentityProviderCaCertPath,
  BaseFormGroupFileUpload                   as FormGroupIdentityProviderCertPath,
  BaseFormGroupInput                        as FormGroupIdentityProviderEntityIdentifier,
  BaseFormGroupFileUpload                   as FormGroupIdentityProviderMetadataPath,
  BaseFormGroupInput                        as FormGroupIdentityToken,
  BaseFormGroupFileUpload                   as FormGroupKeyFile,
  BaseFormGroupChosenOne                    as FormGroupLang,
  BaseFormGroupIntervalUnit                 as FormGroupLocalAccountExpiration,
  BaseFormGroupInputNumber                  as FormGroupLocalAccountLogins,
  BaseFormGroupChosenMultiple               as FormGroupLocalRealm,
  BaseFormGroupInput                        as FormGroupMerchantIdentifier,
  BaseFormGroupTextarea                     as FormGroupMessage,
  BaseFormGroupToggle                       as FormGroupMonitor,
  BaseFormGroupInput                        as FormGroupNasIpAddress,
  BaseFormGroupTextarea                     as FormGroupOptions,
  BaseFormGroupActiveDirectoryPasswordTest  as FormGroupPassword,
  BaseFormGroupInput                        as FormGroupPasswordEmailUpdate,
  BaseFormGroupChosenOne                    as FormGroupPasswordLength,
  BaseFormGroupChosenOne                    as FormGroupPasswordRotation,
  BaseFormGroupFileUpload                   as FormGroupPath,
  BaseFormGroupChosenOne                    as FormGroupPaymentType,
  BaseFormGroupFileUpload                   as FormGroupPaypalCertFile,
  BaseFormGroupPersonMappings               as FormGroupPersonMappings,
  BaseFormGroupInputNumber                  as FormGroupPinCodeLength,
  BaseFormGroupInputNumber                  as FormGroupPort,
  BaseFormGroupInput                        as FormGroupProtectedResourceUrl,
  BaseFormGroupProtocolHostPort             as FormGroupProtocolHostPort,
  BaseFormGroupTextarea                     as FormGroupProxyAddresses,
  BaseFormGroupInput                        as FormGroupPublicClientKey,
  BaseFormGroupInput                        as FormGroupPublishableKey,
  BaseFormGroupInputPassword                as FormGroupRadiusSecret,
  BaseFormGroupInputNumber                  as FormGroupReadTimeout,
  BaseFormGroupChosenMultiple               as FormGroupRealms,
  BaseFormGroupInput                        as FormGroupRedirectUrl,
  BaseFormGroupToggleDisabledEnabled        as FormGroupRegisterOnActivation,
  BaseFormGroupChosenMultiple               as FormGroupRejectRealm,
  BaseFormGroupChosenOne                    as FormGroupScope,
  BaseFormGroupInput                        as FormGroupScopeString,
  BaseFormGroupChosenMultiple               as FormGroupSearchAttributes,
  BaseFormGroupInput                        as FormGroupSearchAttributesAppend,
  BaseFormGroupInputPassword                as FormGroupSecret,
  BaseFormGroupInput                        as FormGroupSecretKey,
  BaseFormGroupToggleDisabledEnabled        as FormGroupSendEmailConfirmation,
  BaseFormGroupServerAddressPort            as FormGroupServerAddressPort,
  BaseFormGroupInput                        as FormGroupServiceFqdn,
  BaseFormGroupInput                        as FormGroupServiceProviderEntityIdentifier,
  BaseFormGroupFileUpload                   as FormGroupServiceProviderCertPath,
  BaseFormGroupFileUpload                   as FormGroupServiceProviderKeyPath,
  BaseFormGroupInput                        as FormGroupSharedSecret,
  BaseFormGroupInput                        as FormGroupSharedSecretDirect,
  BaseFormGroupToggle                       as FormGroupShuffle,
  BaseFormGroupInput                        as FormGroupSite,
  BaseFormGroupIntervalUnit                 as FormGroupSmsActivationTimeout,
  BaseFormGroupChosenMultiple               as FormGroupSmsCarriers,
  BaseFormGroupChosenMultiple               as FormGroupSources,
  BaseFormGroupInput                        as FormGroupSponsorshipBcc,
  BaseFormGroupChosenOne                    as FormGroupStyle,
  BaseFormGroupInput                        as FormGroupTenantIdentifier,
  BaseFormGroupInput                        as FormGroupTerminalGroupIdentifier,
  BaseFormGroupInput                        as FormGroupTerminalIdentifier,
  BaseFormGroupToggle                       as FormGroupTestMode,
  BaseFormGroupInput                        as FormGroupTimeout,
  BaseFormGroupInput                        as FormGroupTransactionKey,
  BaseFormGroupInput                        as FormGroupTwilioPhoneNumber,
  BaseFormGroupToggle                       as FormGroupUseConnector,
  BaseFormGroupInputNumber                  as FormGroupUserGroupsCache,
  BaseFormGroupInput                        as FormGroupUserHeader,
  BaseFormGroupChosenOne                    as FormGroupUsernameAttribute,
  BaseFormGroupInput                        as FormGroupUsernameAttributeString,
  BaseFormGroupToggleNoYes                  as FormGroupValidateSponsor,
  BaseFormGroupChosenOne                    as FormGroupVerify,
  BaseFormGroupInputNumber                  as FormGroupWriteTimeout,
  BaseFormGroupTextarea                     as FormGroupEduroamOptions,
  BaseFormGroupChosenMultiple               as FormGroupEduroamRadiusAuth,
  BaseFormGroupChosenOne                    as FormGroupEduroamRadiusAuthProxyType,
  BaseFormGroupInput                        as FormGroupEduroamOperatorName,

  BaseServices,
  ButtonSamlMetaData,
  TheForm,
  TheView
}
