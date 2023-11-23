# drivr-certificate-client

The drivr-certificate-client is a commandline interface to DRIVR's device certificate management.
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

### Create certificate

Create a certificate for a device running:

    `drivr-certificate-client create certificate -n <devicename> --drivr-api <URL to the DRIVR API>` 

### Fetch certificate

Fetch a requested certificate for a specific device identified by its uuid:

    `drivr-certificate-client fetch certificate --uuid <device uuid> --drivr-api <URL to the DRIVR API>`

## Debugging

Enable debug output via passing `--log-level debug` to `drivr-certificate-client`.
