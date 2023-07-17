# drivr-certificate-client

The drivr-certificate-client is a commandline interface to DRIVR's device certificate managment.
It can be used to:
    - generate an RSA key-pair (private/public)
    - request a certificate for a device from DRIVR
    - fetch a certificate for a specific device

## Building the binary

1. clone the repository
1. run `make`

## Usage

The drivr-certificate-client provides shell completion for bash and zsh.

Enable completion for the current zsh shell run:
    - `source <(drivr-certificate-client completion --shell zsh)`

### Generate key-pair

Generate a private and a public key via:

    `drivr-cert-client create keypair`.

This will create two files `private.key` and `public.key`. 

You can specify different output files with the `--privkey-outfile` and `--pubkey-outfile` arguments.

### Create certificate

Create a certificate for a device running:

    `drivr-certificate-client create certificate -n <devicename> --graphql-api <URL to the GraphQL API> --api-key <API Bearer token>`

API URL and key can also be exported via the environment variables `DRIVR_GRAPHQL_API` and `DRIVR_API_KEY`.

### Fetch certificate

Fetch a requested certificate for a specific device:

    `drivr-certificate-client fetch certificate -n <devicename> --graphql-api <URL to the GraphQL API> --api-key <API Bearer token>`

API URL and key can also be exported via the environment variables `DRIVR_GRAPHQL_API` and `DRIVR_API_KEY`.

## Debugging

Enable debug output globally via the commandline argument `--log-level debug`
