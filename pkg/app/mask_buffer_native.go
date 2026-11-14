//go:build !js

package app

import (
	"reflect"
	"unsafe"

	uimage "github.com/ebitenui/ebitenui/image"
)

var maskBufferType = reflect.TypeFor[*uimage.MaskedRenderBuffer]()

// shareMaskBuffer makes a scroll area or text field draw through *shared, creating it on first use,
// and reports whether it could.
//
// ebitenui gives each of those widgets a private buffer of two images the size of the whole window
// (60 MB at 3840x2160), allocated when first drawn. The panel is rebuilt on every change, and each
// rebuild left the previous pair on the GPU until Go's collector next ran. The collector paces
// itself on Go memory, not GPU memory, so the pairs piled up: a 4K window passed 5 GB within twenty
// rebuilds. The library has no option for this, so the private field is replaced. If a later
// version renames it this returns false and the widget keeps its own buffer.
//
// Widgets may share a buffer only if neither draws inside the other: a draw clears the buffer.
func shareMaskBuffer(w any, shared **uimage.MaskedRenderBuffer) bool {
	v := reflect.ValueOf(w)
	if v.Kind() != reflect.Pointer || v.IsNil() || v.Elem().Kind() != reflect.Struct {
		return false
	}
	f := v.Elem().FieldByName("renderBuf")
	if !f.IsValid() || f.Type() != maskBufferType {
		return false
	}
	if *shared == nil {
		*shared = uimage.NewMaskedRenderBuffer()
	}
	*(**uimage.MaskedRenderBuffer)(unsafe.Pointer(f.UnsafeAddr())) = *shared
	return true
}
