# RPC Codes

This document describes error codes from the service for different custom cases

## Ranges

- `1000` - `2999` Success responses
- `3000` - `3999` Not successful at the moment. Need to wait.
- `4000` - `4999` Failed responses

## Full table with all rpc codes

| Code   | Name                         | Description                                    |
|--------|------------------------------|------------------------------------------------|
| `3000` | Not enough                   | Not enough balance or resources on the wallet  |
| `3001` | Address is taken             | The address is occupied by another transaction |
| `3002` | Max fee exceeded             | Max fee value exceeded                         |
| `3003` | Resource manager is disabled | Resource manager disabled                      |
| `4000` | Blockchain is disabled       | At the moment this blockchain is disabled      |
| `4001` | Zero balance                 | Empty balance on current address               |
