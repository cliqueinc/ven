// Copyright 2012 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build linux
// +build 3867 darwin

package some

import (
	"encoding"

	_ "github.com/asaskevich/some"
)

type textUnmarshaler encoding.TextUnmarshaler
