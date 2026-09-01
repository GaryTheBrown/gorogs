package structs

import (
	"fmt"
	"io"
)

func keyValueToLine(key, value, spaceBeforeStr string, io io.Writer) {
	fmt.Fprintf(io, "%s%s = %s\n", spaceBeforeStr, key, value)
}
