<template>
  <b-container fluid class="p-0">
    <b-tabs v-model="tabIndex" card>

      <!--
        View mode (default)
      -->
      <b-tab :title="$t('View {title} Certificates', { title })" class="p-0">
        <b-card no-body class="m-3">
          <b-card-header>
            <h5 class="mb-0 d-inline">{{ title.value }} {{ $i18n.t('{name} Server Certificate', { name }) }}</h5>
            <b-button v-t="'Generate Signing Request (CSR)'" class="float-right" size="sm" variant="outline-secondary" @click="doShowCsr"/>
            <base-services :value="isModified"
              v-bind="services" class="mt-3 mb-0" variant="info" />
          </b-card-header>
          <base-container-loading v-if="isLoading"
            :title="$i18n.t('Loading Certificate')"
            spin
          />
          <b-container fluid v-else>
            <b-row align-v="center" v-if="isLetsEncrypt">
              <b-col sm="3" class="col-form-label"><icon name="check"/></b-col>
              <b-col sm="9">{{ $t(`Use Let's Encrypt`) }}</b-col>
            </b-row>
            <b-row align-v="center" v-if="isCertKeyMatch">
              <b-col sm="3" class="col-form-label"><icon class="text-success" name="circle"/></b-col>
              <b-col sm="9">{{ $t('Certificate/Key match') }}</b-col>
            </b-row>
            <b-row align-v="center" v-else>
              <b-col sm="3" class="col-form-label"><icon class="text-danger fa-overlap" name="circle"/></b-col>
              <b-col sm="9">{{ $t(`Certificate/Key don't match`) }}</b-col>
            </b-row>
            <b-row align-v="center" v-if="isChainValid">
              <b-col sm="3" class="col-form-label"><icon class="text-success" name="circle"/></b-col>
              <b-col sm="9">{{ $t('Chain is valid') }}</b-col>
            </b-row>
            <b-row align-v="center" v-else>
              <b-col sm="3" class="col-form-label"><icon class="text-danger fa-overlap" name="circle"/></b-col>
              <b-col sm="9">{{ $t('Chain is invalid') }}</b-col>
            </b-row>
            <b-row align-v="baseline" v-for="(value, key) in certificateLocale" :key="key">
              <b-col sm="3" class="col-form-label">{{ key }}</b-col>
                <b-col sm="9" v-if="Array.isArray(value)">
                  <b-badge v-for="(v, k) in value" :key="`${key}-${k}`" class="mr-1" variant="secondary">{{ v }}</b-badge>
                </b-col>
              <b-col sm="9" v-else>{{ value }}</b-col>
            </b-row>
          </b-container>
        </b-card>
        <template v-if="form.info">
          <b-card v-for="(intermediate_ca, index) in intermediateCertificatesLocale" :key="intermediate_ca.serial"
            no-body class="m-3">
            <b-card-header>
              <h4 class="mb-0 d-inline">{{ title }} {{ $t('Intermediate CA certificate') }}</h4>
              <b-badge variant="secondary" class="ml-1">{{ index + 1 }}</b-badge>
            </b-card-header>
            <b-row align-v="center" v-for="(value, key) in intermediate_ca" :key="key">
                <b-col sm="3" class="col-form-label">{{ key }}</b-col>
                <b-col sm="9" v-if="Array.isArray(value)">
                  <b-badge v-for="(v, k) in value" :key="`${key}-${k}`" class="mr-1" variant="secondary">{{ v }}</b-badge>
                </b-col>
              <b-col sm="9" v-else>{{ value }}</b-col>
            </b-row>
          </b-card>
        </template>
        <b-card no-body class="m-3" v-if="isCertificationAuthority">
          <b-card-header>
            <h4 class="mb-0">{{ title }} {{ $t('Certification Authority Certificate(s)') }}</h4>
          </b-card-header>
          <base-container-loading v-if="isLoading"
            :title="$i18n.t('Loading Certification Authority Certificates')"
            spin
          />
          <template v-else>
            <b-container v-for="(ca, index) in certificationAuthorityLocale" :key="index"
              class="mb-3" :class="{ 'border-top': index }" fluid>
              <b-row align-v="center" v-for="(value, key) in ca" :key="key">
                <b-col sm="3" class="col-form-label">{{ key }}</b-col>
                <b-col sm="9" v-if="Array.isArray(value)">
                  <b-badge v-for="(v, k) in value" :key="`${key}-${k}`" class="mr-1" variant="secondary">{{ v }}</b-badge>
                </b-col>
              <b-col sm="9" v-else>{{ value }}</b-col>
              </b-row>
            </b-container>
          </template>
        </b-card>
        <b-card-footer>
          <b-button v-t="'Edit'" @click="tabIndex = 1"/>
        </b-card-footer>
      </b-tab>

      <!--
        Edit mode
      -->
      <b-tab :title="$t('Edit {title} Certificates', { title })" class="p-0">
        <b-form @submit.prevent="onSave" ref="rootRef">
          <base-form
            :form="form.certificate"
            :schema="schema"
            :isLoading="isLoading"
          >
            <form-group-lets-encrypt namespace="lets_encrypt"
              class="no-saas"
              :column-label="$i18n.t(`Use Let's Encrypt`)"
            />

            <!--
              With Let's Encrypt (lets_encrypt: true)
            -->
            <template v-if="form.certificate && form.certificate.lets_encrypt">
              <form-group-lets-encrypt-common-name namespace="common_name"
                :column-label="$i18n.t('Common Name')"
              />

              <form-group-ca v-if="id === 'radius'"
                namespace="ca"
                :column-label="$i18n.t('Certification Authority certificate(s)')"
                rows="6" auto-fit
              />
            </template>

            <!--
              Without Let's Encrypt (lets_encrypt: false)
            -->
            <template v-else>
              <form-group-certificate namespace="certificate"
                :column-label="$i18n.t('{name} Server Certificate', { name })"
                rows="6" auto-fit
              />

              <form-group-find-intermediate-cas v-model="isFindIntermediateCas"
                :column-label="$i18n.t('Find {name} Server intermediate CA(s) automatically', { name })"
              />

              <form-group-intermediate-certification-authorities v-if="!isFindIntermediateCas"
                namespace="intermediate_cas"
                :column-label="$i18n.t('Intermediate CA certificate(s)')"
              />

              <form-group-check-chain namespace="check_chain"
                :column-label="$i18n.t('Validate certificate chain')"
              />

              <form-group-private-key namespace="private_key"
                :column-label="$i18n.t('{name} Server Private Key', { name })"
                rows="6" auto-fit
              />

              <form-group-ca v-if="id === 'radius'"
                namespace="ca"
                :column-label="$i18n.t('Certification Authority certificate(s)')"
                :text="$i18n.t('Trusted client authority certificates including root or intermediate certificates used for EAP-TLS must be added here.')"
                rows="6" auto-fit
              />

            </template>
          </base-form>
        </b-form>
        <b-card-footer>
          <base-form-button-bar
            :is-loading="isLoading"
            :is-saveable="true"
            :is-valid="isValid"
            :form-ref="rootRef"
            @close="tabIndex = 0"
            @reset="onReset"
            @save="onSaveWrapper"
          />
          <base-services :value="isModified"
            v-bind="services" :title="$i18n.t('Warning')" class="mt-3 mb-0" />
        </b-card-footer>
      </b-tab>

      <!--
        CSR modal
      -->
      <the-csr
        v-model="isShowCsr"
        :id="id"
        @hidden="doHideCsr"
      />
    </b-tabs>
  </b-container>
</template>
<script>
import {
  BaseContainerLoading,
  BaseForm,
  BaseFormButtonBar,
  BaseFormGroupToggleFalseTrue as FormGroupFindIntermediateCas,
  BaseServices,
} from '@/components/new/'
import {
  FormGroupCa,
  FormGroupCertificate,
  FormGroupCheckChain,
  FormGroupIntermediateCertificationAuthorities,
  FormGroupLetsEncrypt,
  FormGroupLetsEncryptCommonName,
  FormGroupPrivateKey,
  TheCsr
} from './'

const components = {
  BaseContainerLoading,
  BaseForm,
  BaseFormButtonBar,
  BaseServices,
  FormGroupCa,
  FormGroupCertificate,
  FormGroupCheckChain,
  FormGroupFindIntermediateCas,
  FormGroupIntermediateCertificationAuthorities,
  FormGroupLetsEncrypt,
  FormGroupLetsEncryptCommonName,
  FormGroupPrivateKey,

  TheCsr
}

import { computed, ref, toRefs } from '@vue/composition-api'
import { useForm, useFormProps } from '../_composables/useForm'
import { useViewCollectionItemFixed, useViewCollectionItemFixedProps } from '../../_composables/useViewCollectionItemFixed'
import * as collection from '../_composables/useCollection'

const props = {
  ...useFormProps,
  ...useViewCollectionItemFixedProps,
}

const setup = (props, context) => {

  const {
    id
  } = toRefs(props)
  const name = computed(() => {
    switch(id.value.toLowerCase()) {
      case 'http':
        return 'HTTPs'
        //break
      default:
        return id.value.toUpperCase()
    }
  })

  const {
    rootRef,
    form,
    services,
    title,
    isModified,
    customProps,
    isValid,
    isLoading,
    onReset,
    onSave,
  } = useViewCollectionItemFixed(collection, props, context)

  const {
    schema,
    certificateLocale,
    certificationAuthorityLocale,
    intermediateCertificatesLocale,

    isShowCsr,
    doShowCsr,
    doHideCsr,

    isCertificationAuthority,
    isCertKeyMatch,
    isChainValid,
    isLetsEncrypt,
    isFindIntermediateCas
  } = useForm(form, props, context)

  const { root: { $store } = {} } = context

  const onSaveWrapper = () => {
    onSave()
      .then(() => {
          const { useStore } = collection
          const { getItem } = useStore($store)
          getItem(form.value.certificate)
            .then(item => form.value = item)
            .finally(() => {
              tabIndex.value = 0
            })
      })
  }

  const tabIndex = ref(0)

  return {
    name,
    // useViewCollectionItemFixed
    rootRef,
    form,
    meta: undefined,
    services,
    title,
    isModified,
    customProps,
    isValid,
    isLoading,
    onReset,

    // useForm
    schema,
    certificateLocale,
    certificationAuthorityLocale,
    intermediateCertificatesLocale,
    isShowCsr,
    doShowCsr,
    doHideCsr,
    isCertificationAuthority,
    isCertKeyMatch,
    isChainValid,
    isLetsEncrypt,
    isFindIntermediateCas,

    // custom
    onSaveWrapper,
    tabIndex
  }
}

export default {
  name: 'the-form',
  components,
  props,
  setup
}
</script>
