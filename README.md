# keyswarm
Omni-language ethereum key guesser.

## Installation
You need Go 1.19 installed. See https://go.dev/dl
Other versions of Go may work but have not been tested.
Run:
```
go install github.com/mteam88/keyswarm@v1.0.0
```
Replace `@v1.0.0` with your desired version.

## Setup
Define some Infura API keys in your environment.
Example .env file:
```
INFURA_KEYS=<ONE KEY HERE>,<ANOTHER KEY>,<JUST A COMMA SEPERATING KEYS>,<INFURA.IO EVERYONE>
```
Support for non-infura web3 providers is in development.

## Usage
```
$ keyswarm
```
Enjoy!

### Dev Commands
`docker run --volume /workspaces/keyswarm:/usr/home/keyswarm --workdir /usr/home/keyswarm -it --rm golang`
