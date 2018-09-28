// +build go1.2

package some

import (
	"encoding"

	_ "github.com/asaskevich/govalidator"
)

type textUnmarshaler encoding.TextUnmarshaler
