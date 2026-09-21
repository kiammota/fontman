# fontman

A simple font manager for Linux.

## Features

* List installed fonts
* Search fonts by name or filename
* Install fonts
* Remove fonts
* Inspect font metadata
* Manage user and system fonts

## Usage

```text
fontman list [-u/--user]
fontman install <path>
fontman remove <term> [-g/--global]
fontman grep [-u/--user] <term>
fontman info <term>
fontman help
```

## Examples

```bash
fontman list
fontman list --user

fontman grep "0xProto"

fontman install ./MyFont.ttf

fontman remove "0xProto"
fontman remove "0xProto" --global

fontman info "0xProto"
```

## Supported Formats

* `.ttf`
* `.otf`
* `.zip`

## Installation

```bash
git clone https://github.com/KiamMota/fontman.git
cd fontman
go install
```

## License

MIT
