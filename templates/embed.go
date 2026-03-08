// Package templates embeds all language template directories.
package templates

import "embed"

//go:embed all:go all:python all:rust all:php all:typescript all:javascript
var Embedded embed.FS
