// +build linux
// +build 386 amd64

package some

import (
	"encoding"

	_ "github.com/asaskevich/govalidator"
	_ "github.com/asaskevich/wrong"
)

type textUnmarshaler encoding.TextUnmarshaler
