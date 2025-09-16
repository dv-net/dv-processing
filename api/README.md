# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [processing/client/v1/client.proto](#processing_client_v1_client-proto)
    - [CreateRequest](#processing-client-v1-CreateRequest)
    - [CreateResponse](#processing-client-v1-CreateResponse)
    - [GetCallbackURLRequest](#processing-client-v1-GetCallbackURLRequest)
    - [GetCallbackURLResponse](#processing-client-v1-GetCallbackURLResponse)
    - [UpdateCallbackURLRequest](#processing-client-v1-UpdateCallbackURLRequest)
    - [UpdateCallbackURLResponse](#processing-client-v1-UpdateCallbackURLResponse)
  
    - [ClientService](#processing-client-v1-ClientService)
  
- [processing/common/v1/common.proto](#processing_common_v1_common-proto)
    - [BitcoinAddressType](#processing-common-v1-BitcoinAddressType)
    - [Blockchain](#processing-common-v1-Blockchain)
    - [DogecoinAddressType](#processing-common-v1-DogecoinAddressType)
    - [IncomingWalletType](#processing-common-v1-IncomingWalletType)
    - [LitecoinAddressType](#processing-common-v1-LitecoinAddressType)
    - [TransactionType](#processing-common-v1-TransactionType)
    - [TransferStatus](#processing-common-v1-TransferStatus)
  
- [processing/owner/v1/owner.proto](#processing_owner_v1_owner-proto)
    - [ConfirmTwoFactorAuthRequest](#processing-owner-v1-ConfirmTwoFactorAuthRequest)
    - [ConfirmTwoFactorAuthResponse](#processing-owner-v1-ConfirmTwoFactorAuthResponse)
    - [CreateRequest](#processing-owner-v1-CreateRequest)
    - [CreateResponse](#processing-owner-v1-CreateResponse)
    - [DisableTwoFactorAuthRequest](#processing-owner-v1-DisableTwoFactorAuthRequest)
    - [DisableTwoFactorAuthResponse](#processing-owner-v1-DisableTwoFactorAuthResponse)
    - [GetHotWalletKeysItem](#processing-owner-v1-GetHotWalletKeysItem)
    - [GetHotWalletKeysRequest](#processing-owner-v1-GetHotWalletKeysRequest)
    - [GetHotWalletKeysResponse](#processing-owner-v1-GetHotWalletKeysResponse)
    - [GetPrivateKeysRequest](#processing-owner-v1-GetPrivateKeysRequest)
    - [GetPrivateKeysResponse](#processing-owner-v1-GetPrivateKeysResponse)
    - [GetPrivateKeysResponse.KeysEntry](#processing-owner-v1-GetPrivateKeysResponse-KeysEntry)
    - [GetSeedsRequest](#processing-owner-v1-GetSeedsRequest)
    - [GetSeedsResponse](#processing-owner-v1-GetSeedsResponse)
    - [GetTwoFactorAuthDataRequest](#processing-owner-v1-GetTwoFactorAuthDataRequest)
    - [GetTwoFactorAuthDataResponse](#processing-owner-v1-GetTwoFactorAuthDataResponse)
    - [KeyPair](#processing-owner-v1-KeyPair)
    - [KeyPairSequence](#processing-owner-v1-KeyPairSequence)
    - [PrivateKeyItem](#processing-owner-v1-PrivateKeyItem)
    - [ValidateTwoFactorTokenRequest](#processing-owner-v1-ValidateTwoFactorTokenRequest)
    - [ValidateTwoFactorTokenResponse](#processing-owner-v1-ValidateTwoFactorTokenResponse)
  
    - [OwnerService](#processing-owner-v1-OwnerService)
  
- [processing/system/v1/system.proto](#processing_system_v1_system-proto)
    - [CheckNewVersionRequest](#processing-system-v1-CheckNewVersionRequest)
    - [CheckNewVersionResponse](#processing-system-v1-CheckNewVersionResponse)
    - [GetLastLogsRequest](#processing-system-v1-GetLastLogsRequest)
    - [GetLastLogsResponse](#processing-system-v1-GetLastLogsResponse)
    - [InfoRequest](#processing-system-v1-InfoRequest)
    - [InfoResponse](#processing-system-v1-InfoResponse)
    - [LogEntry](#processing-system-v1-LogEntry)
    - [UpdateToNewVersionRequest](#processing-system-v1-UpdateToNewVersionRequest)
    - [UpdateToNewVersionResponse](#processing-system-v1-UpdateToNewVersionResponse)
  
    - [SystemService](#processing-system-v1-SystemService)
  
- [processing/transfer/v1/transfer.proto](#processing_transfer_v1_transfer-proto)
    - [CreateRequest](#processing-transfer-v1-CreateRequest)
    - [CreateResponse](#processing-transfer-v1-CreateResponse)
    - [GetByRequestIDRequest](#processing-transfer-v1-GetByRequestIDRequest)
    - [GetByRequestIDResponse](#processing-transfer-v1-GetByRequestIDResponse)
    - [Transfer](#processing-transfer-v1-Transfer)
    - [TransferTransaction](#processing-transfer-v1-TransferTransaction)
  
    - [Status](#processing-transfer-v1-Status)
    - [TransferTransactionStatus](#processing-transfer-v1-TransferTransactionStatus)
    - [TransferTransactionType](#processing-transfer-v1-TransferTransactionType)
  
    - [TransferService](#processing-transfer-v1-TransferService)
  
- [processing/wallet/v1/wallets.proto](#processing_wallet_v1_wallets-proto)
    - [Asset](#processing-wallet-v1-Asset)
    - [Assets](#processing-wallet-v1-Assets)
    - [AttachOwnerColdWalletsRequest](#processing-wallet-v1-AttachOwnerColdWalletsRequest)
    - [AttachOwnerColdWalletsResponse](#processing-wallet-v1-AttachOwnerColdWalletsResponse)
    - [BlockchainAdditionalData](#processing-wallet-v1-BlockchainAdditionalData)
    - [BlockchainAdditionalData.TronData](#processing-wallet-v1-BlockchainAdditionalData-TronData)
    - [CreateOwnerHotWalletRequest](#processing-wallet-v1-CreateOwnerHotWalletRequest)
    - [CreateOwnerHotWalletResponse](#processing-wallet-v1-CreateOwnerHotWalletResponse)
    - [GetOwnerColdWalletsRequest](#processing-wallet-v1-GetOwnerColdWalletsRequest)
    - [GetOwnerColdWalletsResponse](#processing-wallet-v1-GetOwnerColdWalletsResponse)
    - [GetOwnerHotWalletsRequest](#processing-wallet-v1-GetOwnerHotWalletsRequest)
    - [GetOwnerHotWalletsResponse](#processing-wallet-v1-GetOwnerHotWalletsResponse)
    - [GetOwnerHotWalletsResponse.HotAddress](#processing-wallet-v1-GetOwnerHotWalletsResponse-HotAddress)
    - [GetOwnerProcessingWalletsRequest](#processing-wallet-v1-GetOwnerProcessingWalletsRequest)
    - [GetOwnerProcessingWalletsResponse](#processing-wallet-v1-GetOwnerProcessingWalletsResponse)
    - [MarkDirtyHotWalletRequest](#processing-wallet-v1-MarkDirtyHotWalletRequest)
    - [MarkDirtyHotWalletResponse](#processing-wallet-v1-MarkDirtyHotWalletResponse)
    - [WalletPreview](#processing-wallet-v1-WalletPreview)
  
    - [WalletService](#processing-wallet-v1-WalletService)
  
- [Scalar Value Types](#scalar-value-types)



<a name="processing_client_v1_client-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## processing/client/v1/client.proto



<a name="processing-client-v1-CreateRequest"></a>

### CreateRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| callback_url | [string](#string) |  |  |
| backend_ip | [string](#string) | optional |  |
| merchant_domain | [string](#string) | optional |  |
| backend_version | [string](#string) |  |  |






<a name="processing-client-v1-CreateResponse"></a>

### CreateResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| client_id | [string](#string) |  |  |
| client_key | [string](#string) |  |  |
| admin_secret_key | [string](#string) |  |  |






<a name="processing-client-v1-GetCallbackURLRequest"></a>

### GetCallbackURLRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| client_id | [string](#string) |  |  |






<a name="processing-client-v1-GetCallbackURLResponse"></a>

### GetCallbackURLResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| callback_url | [string](#string) |  |  |






<a name="processing-client-v1-UpdateCallbackURLRequest"></a>

### UpdateCallbackURLRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| client_id | [string](#string) |  |  |
| callback_url | [string](#string) |  |  |






<a name="processing-client-v1-UpdateCallbackURLResponse"></a>

### UpdateCallbackURLResponse






 

 

 


<a name="processing-client-v1-ClientService"></a>

### ClientService
Service which interacts with client

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| Create | [CreateRequest](#processing-client-v1-CreateRequest) | [CreateResponse](#processing-client-v1-CreateResponse) | Create client |
| UpdateCallbackURL | [UpdateCallbackURLRequest](#processing-client-v1-UpdateCallbackURLRequest) | [UpdateCallbackURLResponse](#processing-client-v1-UpdateCallbackURLResponse) | Change merchant callback url |
| GetCallbackURL | [GetCallbackURLRequest](#processing-client-v1-GetCallbackURLRequest) | [GetCallbackURLResponse](#processing-client-v1-GetCallbackURLResponse) | Get merchant callback url |

 



<a name="processing_common_v1_common-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## processing/common/v1/common.proto


 


<a name="processing-common-v1-BitcoinAddressType"></a>

### BitcoinAddressType


| Name | Number | Description |
| ---- | ------ | ----------- |
| BITCOIN_ADDRESS_TYPE_UNSPECIFIED | 0 |  |
| BITCOIN_ADDRESS_TYPE_P2PKH | 1 | Legacy |
| BITCOIN_ADDRESS_TYPE_P2SH | 2 | SegWit |
| BITCOIN_ADDRESS_TYPE_SEGWIT | 3 | Native SegWit or Bech32 |
| BITCOIN_ADDRESS_TYPE_P2TR | 4 | Taproot address or Bech32m |



<a name="processing-common-v1-Blockchain"></a>

### Blockchain


| Name | Number | Description |
| ---- | ------ | ----------- |
| BLOCKCHAIN_UNSPECIFIED | 0 |  |
| BLOCKCHAIN_TRON | 1 |  |
| BLOCKCHAIN_BITCOIN | 2 |  |
| BLOCKCHAIN_ETHEREUM | 3 |  |
| BLOCKCHAIN_LITECOIN | 4 |  |
| BLOCKCHAIN_BITCOINCASH | 5 |  |
| BLOCKCHAIN_BINANCE_SMART_CHAIN | 6 |  |
| BLOCKCHAIN_POLYGON | 7 |  |
| BLOCKCHAIN_ARBITRUM | 8 |  |
| BLOCKCHAIN_OPTIMISM | 9 |  |
| BLOCKCHAIN_LINEA | 10 |  |
| BLOCKCHAIN_SOLANA | 11 |  |
| BLOCKCHAIN_MONERO | 12 |  |
| BLOCKCHAIN_DOGECOIN | 13 |  |
| BLOCKCHAIN_TON | 14 |  |



<a name="processing-common-v1-DogecoinAddressType"></a>

### DogecoinAddressType


| Name | Number | Description |
| ---- | ------ | ----------- |
| DOGECOIN_ADDRESS_TYPE_UNSPECIFIED | 0 |  |
| DOGECOIN_ADDRESS_TYPE_P2PKH | 1 | Legacy |



<a name="processing-common-v1-IncomingWalletType"></a>

### IncomingWalletType


| Name | Number | Description |
| ---- | ------ | ----------- |
| INCOMING_WALLET_TYPE_UNSPECIFIED | 0 |  |
| INCOMING_WALLET_TYPE_HOT | 1 |  |
| INCOMING_WALLET_TYPE_PROCESSING | 2 |  |



<a name="processing-common-v1-LitecoinAddressType"></a>

### LitecoinAddressType


| Name | Number | Description |
| ---- | ------ | ----------- |
| LITECOIN_ADDRESS_TYPE_UNSPECIFIED | 0 |  |
| LITECOIN_ADDRESS_TYPE_P2PKH | 1 | Legacy |
| LITECOIN_ADDRESS_TYPE_P2SH | 2 | SegWit |
| LITECOIN_ADDRESS_TYPE_SEGWIT | 3 | Native SegWit or Bech32 |
| LITECOIN_ADDRESS_TYPE_P2TR | 4 | Taproot address or Bech32m |



<a name="processing-common-v1-TransactionType"></a>

### TransactionType


| Name | Number | Description |
| ---- | ------ | ----------- |
| TRANSACTION_TYPE_UNSPECIFIED | 0 |  |
| TRANSACTION_TYPE_TRANSFER | 1 |  |
| TRANSACTION_TYPE_DEPOSIT | 2 |  |



<a name="processing-common-v1-TransferStatus"></a>

### TransferStatus


| Name | Number | Description |
| ---- | ------ | ----------- |
| TRANSFER_STATUS_UNSPECIFIED | 0 |  |
| TRANSFER_STATUS_ACCEPTED | 1 |  |
| TRANSFER_STATUS_SUCCESS | 2 |  |
| TRANSFER_STATUS_FAILED | 3 |  |


 

 

 



<a name="processing_owner_v1_owner-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## processing/owner/v1/owner.proto



<a name="processing-owner-v1-ConfirmTwoFactorAuthRequest"></a>

### ConfirmTwoFactorAuthRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| owner_id | [string](#string) |  |  |
| totp | [string](#string) |  |  |






<a name="processing-owner-v1-ConfirmTwoFactorAuthResponse"></a>

### ConfirmTwoFactorAuthResponse







<a name="processing-owner-v1-CreateRequest"></a>

### CreateRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| client_id | [string](#string) |  |  |
| external_id | [string](#string) |  | External id of store |
| mnemonic | [string](#string) |  |  |






<a name="processing-owner-v1-CreateResponse"></a>

### CreateResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |






<a name="processing-owner-v1-DisableTwoFactorAuthRequest"></a>

### DisableTwoFactorAuthRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| owner_id | [string](#string) |  |  |
| totp | [string](#string) |  |  |






<a name="processing-owner-v1-DisableTwoFactorAuthResponse"></a>

### DisableTwoFactorAuthResponse







<a name="processing-owner-v1-GetHotWalletKeysItem"></a>

### GetHotWalletKeysItem



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [processing.common.v1.Blockchain](#processing-common-v1-Blockchain) |  |  |
| items | [PrivateKeyItem](#processing-owner-v1-PrivateKeyItem) | repeated |  |






<a name="processing-owner-v1-GetHotWalletKeysRequest"></a>

### GetHotWalletKeysRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| owner_id | [string](#string) |  |  |
| otp | [string](#string) |  |  |
| wallet_addresses | [string](#string) | repeated |  |
| excluded_wallet_addresses | [string](#string) | repeated |  |






<a name="processing-owner-v1-GetHotWalletKeysResponse"></a>

### GetHotWalletKeysResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| entries | [GetHotWalletKeysItem](#processing-owner-v1-GetHotWalletKeysItem) | repeated |  |






<a name="processing-owner-v1-GetPrivateKeysRequest"></a>

### GetPrivateKeysRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| owner_id | [string](#string) |  |  |
| totp | [string](#string) |  |  |






<a name="processing-owner-v1-GetPrivateKeysResponse"></a>

### GetPrivateKeysResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| keys | [GetPrivateKeysResponse.KeysEntry](#processing-owner-v1-GetPrivateKeysResponse-KeysEntry) | repeated |  |






<a name="processing-owner-v1-GetPrivateKeysResponse-KeysEntry"></a>

### GetPrivateKeysResponse.KeysEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [KeyPairSequence](#processing-owner-v1-KeyPairSequence) |  |  |






<a name="processing-owner-v1-GetSeedsRequest"></a>

### GetSeedsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| owner_id | [string](#string) |  |  |
| totp | [string](#string) |  |  |






<a name="processing-owner-v1-GetSeedsResponse"></a>

### GetSeedsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| mnemonic | [string](#string) |  |  |
| pass_phrase | [string](#string) |  |  |






<a name="processing-owner-v1-GetTwoFactorAuthDataRequest"></a>

### GetTwoFactorAuthDataRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| owner_id | [string](#string) |  |  |






<a name="processing-owner-v1-GetTwoFactorAuthDataResponse"></a>

### GetTwoFactorAuthDataResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| secret | [string](#string) | optional |  |
| is_confirmed | [bool](#bool) |  |  |






<a name="processing-owner-v1-KeyPair"></a>

### KeyPair



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| public_key | [string](#string) |  |  |
| private_key | [string](#string) |  |  |
| address | [string](#string) |  |  |
| kind | [string](#string) |  |  |






<a name="processing-owner-v1-KeyPairSequence"></a>

### KeyPairSequence



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| pairs | [KeyPair](#processing-owner-v1-KeyPair) | repeated |  |






<a name="processing-owner-v1-PrivateKeyItem"></a>

### PrivateKeyItem



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| address | [string](#string) |  |  |
| public_key | [string](#string) |  |  |
| private_key | [string](#string) |  |  |






<a name="processing-owner-v1-ValidateTwoFactorTokenRequest"></a>

### ValidateTwoFactorTokenRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| owner_id | [string](#string) |  |  |
| totp | [string](#string) |  |  |






<a name="processing-owner-v1-ValidateTwoFactorTokenResponse"></a>

### ValidateTwoFactorTokenResponse






 

 

 


<a name="processing-owner-v1-OwnerService"></a>

### OwnerService
Service which interacts with owner

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| Create | [CreateRequest](#processing-owner-v1-CreateRequest) | [CreateResponse](#processing-owner-v1-CreateResponse) | Create owner of client (creates processing wallet as side effect) |
| GetSeeds | [GetSeedsRequest](#processing-owner-v1-GetSeedsRequest) | [GetSeedsResponse](#processing-owner-v1-GetSeedsResponse) | Get owner mnemonic phrases |
| GetPrivateKeys | [GetPrivateKeysRequest](#processing-owner-v1-GetPrivateKeysRequest) | [GetPrivateKeysResponse](#processing-owner-v1-GetPrivateKeysResponse) | Get owner private keys (only hot,processing) |
| GetHotWalletKeys | [GetHotWalletKeysRequest](#processing-owner-v1-GetHotWalletKeysRequest) | [GetHotWalletKeysResponse](#processing-owner-v1-GetHotWalletKeysResponse) | Get owner hot wallet keys |
| ConfirmTwoFactorAuth | [ConfirmTwoFactorAuthRequest](#processing-owner-v1-ConfirmTwoFactorAuthRequest) | [ConfirmTwoFactorAuthResponse](#processing-owner-v1-ConfirmTwoFactorAuthResponse) | Confirm owner two auth |
| DisableTwoFactorAuth | [DisableTwoFactorAuthRequest](#processing-owner-v1-DisableTwoFactorAuthRequest) | [DisableTwoFactorAuthResponse](#processing-owner-v1-DisableTwoFactorAuthResponse) | Enable or disable owners two auth |
| GetTwoFactorAuthData | [GetTwoFactorAuthDataRequest](#processing-owner-v1-GetTwoFactorAuthDataRequest) | [GetTwoFactorAuthDataResponse](#processing-owner-v1-GetTwoFactorAuthDataResponse) | Get owner 2fa status data |
| ValidateTwoFactorToken | [ValidateTwoFactorTokenRequest](#processing-owner-v1-ValidateTwoFactorTokenRequest) | [ValidateTwoFactorTokenResponse](#processing-owner-v1-ValidateTwoFactorTokenResponse) | Validate 2fa token |

 



<a name="processing_system_v1_system-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## processing/system/v1/system.proto



<a name="processing-system-v1-CheckNewVersionRequest"></a>

### CheckNewVersionRequest







<a name="processing-system-v1-CheckNewVersionResponse"></a>

### CheckNewVersionResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| installed_version | [string](#string) |  |  |
| available_version | [string](#string) |  |  |
| need_for_update | [bool](#bool) |  |  |






<a name="processing-system-v1-GetLastLogsRequest"></a>

### GetLastLogsRequest







<a name="processing-system-v1-GetLastLogsResponse"></a>

### GetLastLogsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| logs | [LogEntry](#processing-system-v1-LogEntry) | repeated |  |






<a name="processing-system-v1-InfoRequest"></a>

### InfoRequest







<a name="processing-system-v1-InfoResponse"></a>

### InfoResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| version | [string](#string) |  |  |
| commit | [string](#string) |  |  |






<a name="processing-system-v1-LogEntry"></a>

### LogEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| level | [string](#string) |  |  |
| message | [string](#string) |  |  |






<a name="processing-system-v1-UpdateToNewVersionRequest"></a>

### UpdateToNewVersionRequest







<a name="processing-system-v1-UpdateToNewVersionResponse"></a>

### UpdateToNewVersionResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [string](#string) |  |  |





 

 

 


<a name="processing-system-v1-SystemService"></a>

### SystemService
Service which provides system information

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| Info | [InfoRequest](#processing-system-v1-InfoRequest) | [InfoResponse](#processing-system-v1-InfoResponse) | System info (version etc) |
| CheckNewVersion | [CheckNewVersionRequest](#processing-system-v1-CheckNewVersionRequest) | [CheckNewVersionResponse](#processing-system-v1-CheckNewVersionResponse) | Check new version from updater |
| UpdateToNewVersion | [UpdateToNewVersionRequest](#processing-system-v1-UpdateToNewVersionRequest) | [UpdateToNewVersionResponse](#processing-system-v1-UpdateToNewVersionResponse) | Update Processing from updater |
| GetLastLogs | [GetLastLogsRequest](#processing-system-v1-GetLastLogsRequest) | [GetLastLogsResponse](#processing-system-v1-GetLastLogsResponse) | Get last memory logs |

 



<a name="processing_transfer_v1_transfer-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## processing/transfer/v1/transfer.proto



<a name="processing-transfer-v1-CreateRequest"></a>

### CreateRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| owner_id | [string](#string) |  |  |
| request_id | [string](#string) |  |  |
| blockchain | [processing.common.v1.Blockchain](#processing-common-v1-Blockchain) |  |  |
| from_addresses | [string](#string) | repeated |  |
| to_addresses | [string](#string) | repeated |  |
| asset_identifier | [string](#string) |  |  |
| whole_amount | [bool](#bool) |  | withdraw the entire amount from the wallet |
| amount | [string](#string) | optional |  |
| kind | [string](#string) | optional | delegate / burn / etc... |
| fee | [string](#string) | optional |  |
| fee_max | [string](#string) | optional |  |






<a name="processing-transfer-v1-CreateResponse"></a>

### CreateResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| item | [Transfer](#processing-transfer-v1-Transfer) |  |  |






<a name="processing-transfer-v1-GetByRequestIDRequest"></a>

### GetByRequestIDRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| request_id | [string](#string) |  |  |






<a name="processing-transfer-v1-GetByRequestIDResponse"></a>

### GetByRequestIDResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| item | [Transfer](#processing-transfer-v1-Transfer) |  |  |






<a name="processing-transfer-v1-Transfer"></a>

### Transfer
Transfer


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| status | [Status](#processing-transfer-v1-Status) |  |  |
| owner_id | [string](#string) |  |  |
| request_id | [string](#string) |  |  |
| blockchain | [processing.common.v1.Blockchain](#processing-common-v1-Blockchain) |  |  |
| from_addresses | [string](#string) | repeated |  |
| to_addresses | [string](#string) | repeated |  |
| asset_identifier | [string](#string) |  |  |
| kind | [string](#string) | optional | used for tron transfers: burntrx, resources. for other blockchains must be empty |
| whole_amount | [bool](#bool) |  |  |
| amount | [string](#string) | optional |  |
| fee | [string](#string) | optional |  |
| fee_max | [string](#string) | optional |  |
| tx_hash | [string](#string) | optional |  |
| error_message | [string](#string) | optional |  |
| created_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| updated_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) | optional |  |
| state_data | [google.protobuf.Struct](#google-protobuf-Struct) |  |  |
| workflow_snapshot | [google.protobuf.Struct](#google-protobuf-Struct) |  |  |
| transactions | [TransferTransaction](#processing-transfer-v1-TransferTransaction) | repeated | List of system transactions associated with the transfer, sorted by created_at |






<a name="processing-transfer-v1-TransferTransaction"></a>

### TransferTransaction
Transfer transaction


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | UUID |
| transfer_id | [string](#string) |  | UUID |
| tx_hash | [string](#string) |  |  |
| bandwidth_amount | [string](#string) |  | Decimal |
| energy_amount | [string](#string) |  | Decimal |
| native_token_amount | [string](#string) |  | Decimal |
| native_token_fee | [string](#string) |  | Decimal |
| tx_type | [TransferTransactionType](#processing-transfer-v1-TransferTransactionType) |  |  |
| status | [TransferTransactionStatus](#processing-transfer-v1-TransferTransactionStatus) |  |  |
| step | [string](#string) |  |  |
| created_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| updated_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |





 


<a name="processing-transfer-v1-Status"></a>

### Status
Transfer status

| Name | Number | Description |
| ---- | ------ | ----------- |
| STATUS_UNSPECIFIED | 0 |  |
| STATUS_NEW | 1 |  |
| STATUS_PENDING | 2 |  |
| STATUS_PROCESSING | 3 |  |
| STATUS_IN_MEMPOOL | 4 |  |
| STATUS_UNCONFIRMED | 5 |  |
| STATUS_COMPLETED | 6 |  |
| STATUS_FAILED | 7 |  |
| STATUS_FROZEN | 8 |  |



<a name="processing-transfer-v1-TransferTransactionStatus"></a>

### TransferTransactionStatus
Transfer transaction status

| Name | Number | Description |
| ---- | ------ | ----------- |
| TRANSFER_TRANSACTION_STATUS_UNSPECIFIED | 0 |  |
| TRANSFER_TRANSACTION_STATUS_PENDING | 1 |  |
| TRANSFER_TRANSACTION_STATUS_UNCONFIRMED | 2 |  |
| TRANSFER_TRANSACTION_STATUS_CONFIRMED | 3 |  |
| TRANSFER_TRANSACTION_STATUS_FAILED | 4 |  |



<a name="processing-transfer-v1-TransferTransactionType"></a>

### TransferTransactionType
Transfer transaction type

| Name | Number | Description |
| ---- | ------ | ----------- |
| TRANSFER_TRANSACTION_TYPE_UNSPECIFIED | 0 |  |
| TRANSFER_TRANSACTION_TYPE_TRANSFER | 1 |  |
| TRANSFER_TRANSACTION_TYPE_DELEGATE | 2 |  |
| TRANSFER_TRANSACTION_TYPE_RECLAIM | 3 |  |
| TRANSFER_TRANSACTION_TYPE_SEND_BURN_BASE_ASSET | 4 |  |
| TRANSFER_TRANSACTION_TYPE_ACCOUNT_ACTIVATION | 5 |  |


 

 


<a name="processing-transfer-v1-TransferService"></a>

### TransferService
Service which interacts with transfers

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| Create | [CreateRequest](#processing-transfer-v1-CreateRequest) | [CreateResponse](#processing-transfer-v1-CreateResponse) | Create a new transfer |
| GetByRequestID | [GetByRequestIDRequest](#processing-transfer-v1-GetByRequestIDRequest) | [GetByRequestIDResponse](#processing-transfer-v1-GetByRequestIDResponse) | Get transfer by request ID |

 



<a name="processing_wallet_v1_wallets-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## processing/wallet/v1/wallets.proto



<a name="processing-wallet-v1-Asset"></a>

### Asset



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| identity | [string](#string) |  |  |
| amount | [string](#string) |  |  |






<a name="processing-wallet-v1-Assets"></a>

### Assets



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| asset | [Asset](#processing-wallet-v1-Asset) | repeated |  |






<a name="processing-wallet-v1-AttachOwnerColdWalletsRequest"></a>

### AttachOwnerColdWalletsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| owner_id | [string](#string) |  |  |
| blockchain | [processing.common.v1.Blockchain](#processing-common-v1-Blockchain) |  |  |
| totp | [string](#string) |  |  |
| addresses | [string](#string) | repeated |  |






<a name="processing-wallet-v1-AttachOwnerColdWalletsResponse"></a>

### AttachOwnerColdWalletsResponse







<a name="processing-wallet-v1-BlockchainAdditionalData"></a>

### BlockchainAdditionalData



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| tron_data | [BlockchainAdditionalData.TronData](#processing-wallet-v1-BlockchainAdditionalData-TronData) | optional |  |






<a name="processing-wallet-v1-BlockchainAdditionalData-TronData"></a>

### BlockchainAdditionalData.TronData



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| available_energy_for_use | [string](#string) |  |  |
| total_energy | [string](#string) |  |  |
| available_bandwidth_for_use | [string](#string) |  |  |
| total_bandwidth | [string](#string) |  |  |
| stacked_trx | [string](#string) |  |  |
| stacked_energy | [string](#string) |  |  |
| stacked_bandwidth | [string](#string) |  |  |
| stacked_energy_trx | [string](#string) |  |  |
| stacked_bandwidth_trx | [string](#string) |  |  |
| total_used_bandwidth | [string](#string) |  |  |
| total_used_energy | [string](#string) |  |  |






<a name="processing-wallet-v1-CreateOwnerHotWalletRequest"></a>

### CreateOwnerHotWalletRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| owner_id | [string](#string) |  |  |
| blockchain | [processing.common.v1.Blockchain](#processing-common-v1-Blockchain) |  |  |
| external_wallet_id | [string](#string) |  | a store customer who has been given a hot wallet for payment |
| bitcoin_address_type | [processing.common.v1.BitcoinAddressType](#processing-common-v1-BitcoinAddressType) | optional |  |
| litecoin_address_type | [processing.common.v1.LitecoinAddressType](#processing-common-v1-LitecoinAddressType) | optional |  |
| dogecoin_address_type | [processing.common.v1.DogecoinAddressType](#processing-common-v1-DogecoinAddressType) | optional |  |






<a name="processing-wallet-v1-CreateOwnerHotWalletResponse"></a>

### CreateOwnerHotWalletResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| address | [string](#string) |  |  |






<a name="processing-wallet-v1-GetOwnerColdWalletsRequest"></a>

### GetOwnerColdWalletsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| owner_id | [string](#string) |  |  |
| blockchain | [processing.common.v1.Blockchain](#processing-common-v1-Blockchain) | optional |  |






<a name="processing-wallet-v1-GetOwnerColdWalletsResponse"></a>

### GetOwnerColdWalletsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| items | [WalletPreview](#processing-wallet-v1-WalletPreview) | repeated |  |






<a name="processing-wallet-v1-GetOwnerHotWalletsRequest"></a>

### GetOwnerHotWalletsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| owner_id | [string](#string) |  |  |
| blockchain | [processing.common.v1.Blockchain](#processing-common-v1-Blockchain) |  |  |
| external_wallet_id | [string](#string) | optional | a store customer who has been given a hot wallet for payment |
| bitcoin_address_type | [processing.common.v1.BitcoinAddressType](#processing-common-v1-BitcoinAddressType) | optional |  |
| litecoin_address_type | [processing.common.v1.LitecoinAddressType](#processing-common-v1-LitecoinAddressType) | optional |  |






<a name="processing-wallet-v1-GetOwnerHotWalletsResponse"></a>

### GetOwnerHotWalletsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| addresses | [GetOwnerHotWalletsResponse.HotAddress](#processing-wallet-v1-GetOwnerHotWalletsResponse-HotAddress) | repeated |  |






<a name="processing-wallet-v1-GetOwnerHotWalletsResponse-HotAddress"></a>

### GetOwnerHotWalletsResponse.HotAddress



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| address | [string](#string) |  |  |
| external_wallet_id | [string](#string) |  |  |






<a name="processing-wallet-v1-GetOwnerProcessingWalletsRequest"></a>

### GetOwnerProcessingWalletsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| owner_id | [string](#string) |  |  |
| blockchain | [processing.common.v1.Blockchain](#processing-common-v1-Blockchain) | optional |  |
| tiny | [bool](#bool) | optional |  |






<a name="processing-wallet-v1-GetOwnerProcessingWalletsResponse"></a>

### GetOwnerProcessingWalletsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| items | [WalletPreview](#processing-wallet-v1-WalletPreview) | repeated |  |






<a name="processing-wallet-v1-MarkDirtyHotWalletRequest"></a>

### MarkDirtyHotWalletRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| owner_id | [string](#string) |  |  |
| blockchain | [processing.common.v1.Blockchain](#processing-common-v1-Blockchain) |  |  |
| address | [string](#string) |  |  |






<a name="processing-wallet-v1-MarkDirtyHotWalletResponse"></a>

### MarkDirtyHotWalletResponse







<a name="processing-wallet-v1-WalletPreview"></a>

### WalletPreview



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| address | [string](#string) |  |  |
| blockchain | [processing.common.v1.Blockchain](#processing-common-v1-Blockchain) |  |  |
| assets | [Assets](#processing-wallet-v1-Assets) | optional |  |
| blockchain_additional_data | [BlockchainAdditionalData](#processing-wallet-v1-BlockchainAdditionalData) | optional |  |





 

 

 


<a name="processing-wallet-v1-WalletService"></a>

### WalletService
Service which interacts with wallets

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetOwnerHotWallets | [GetOwnerHotWalletsRequest](#processing-wallet-v1-GetOwnerHotWalletsRequest) | [GetOwnerHotWalletsResponse](#processing-wallet-v1-GetOwnerHotWalletsResponse) | Get owner hot wallets |
| GetOwnerColdWallets | [GetOwnerColdWalletsRequest](#processing-wallet-v1-GetOwnerColdWalletsRequest) | [GetOwnerColdWalletsResponse](#processing-wallet-v1-GetOwnerColdWalletsResponse) | Get owner cold active wallet list |
| GetOwnerProcessingWallets | [GetOwnerProcessingWalletsRequest](#processing-wallet-v1-GetOwnerProcessingWalletsRequest) | [GetOwnerProcessingWalletsResponse](#processing-wallet-v1-GetOwnerProcessingWalletsResponse) | Get owner processing wallets |
| AttachOwnerColdWallets | [AttachOwnerColdWalletsRequest](#processing-wallet-v1-AttachOwnerColdWalletsRequest) | [AttachOwnerColdWalletsResponse](#processing-wallet-v1-AttachOwnerColdWalletsResponse) | Attach owner cold wallets |
| MarkDirtyHotWallet | [MarkDirtyHotWalletRequest](#processing-wallet-v1-MarkDirtyHotWalletRequest) | [MarkDirtyHotWalletResponse](#processing-wallet-v1-MarkDirtyHotWalletResponse) | Mark a dirty hot wallet |
| CreateOwnerHotWallet | [CreateOwnerHotWalletRequest](#processing-wallet-v1-CreateOwnerHotWalletRequest) | [CreateOwnerHotWalletResponse](#processing-wallet-v1-CreateOwnerHotWalletResponse) | Create owner hot wallet |

 



## Scalar Value Types

| .proto Type | Notes | C++ | Java | Python | Go | C# | PHP | Ruby |
| ----------- | ----- | --- | ---- | ------ | -- | -- | --- | ---- |
| <a name="double" /> double |  | double | double | float | float64 | double | float | Float |
| <a name="float" /> float |  | float | float | float | float32 | float | float | Float |
| <a name="int32" /> int32 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint32 instead. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="int64" /> int64 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint64 instead. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="uint32" /> uint32 | Uses variable-length encoding. | uint32 | int | int/long | uint32 | uint | integer | Bignum or Fixnum (as required) |
| <a name="uint64" /> uint64 | Uses variable-length encoding. | uint64 | long | int/long | uint64 | ulong | integer/string | Bignum or Fixnum (as required) |
| <a name="sint32" /> sint32 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int32s. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="sint64" /> sint64 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int64s. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="fixed32" /> fixed32 | Always four bytes. More efficient than uint32 if values are often greater than 2^28. | uint32 | int | int | uint32 | uint | integer | Bignum or Fixnum (as required) |
| <a name="fixed64" /> fixed64 | Always eight bytes. More efficient than uint64 if values are often greater than 2^56. | uint64 | long | int/long | uint64 | ulong | integer/string | Bignum |
| <a name="sfixed32" /> sfixed32 | Always four bytes. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="sfixed64" /> sfixed64 | Always eight bytes. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="bool" /> bool |  | bool | boolean | boolean | bool | bool | boolean | TrueClass/FalseClass |
| <a name="string" /> string | A string must always contain UTF-8 encoded or 7-bit ASCII text. | string | String | str/unicode | string | string | string | String (UTF-8) |
| <a name="bytes" /> bytes | May contain any arbitrary sequence of bytes. | string | ByteString | str | []byte | ByteString | string | String (ASCII-8BIT) |

