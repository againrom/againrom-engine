package game

import (
	"fmt"

	"againrom/pkg/data"
	"againrom/pkg/formats/reg"
	"againrom/pkg/render/terrain"
)

// REG-OBJ-047

// ObjectRegistry is the ADDRESS the object classes are loaded from: the graphics
// container's identity segment and the entry inside it, folded and
// forward-separated as every address is compared.
//
// The identity segment is part of the name because ONE string now identifies
// this registry across every container an install ships, so this package
// states which container its classes come from instead of leaving its caller
// to know it out of band.
const ObjectRegistry = graphicsPrefix + "objects/objects.reg"

// ObjectCatalog is the object registry read once: each class by placement
// byte, with its name.
type ObjectCatalog struct{ classes *data.ObjectClasses }

// LoadObjectCatalog reads the object registry of src. A read error comes back
// unwrapped (errors.Is with fs.ErrNotExist works); a parse error names the
// registry.
func LoadObjectCatalog(src terrain.EntrySource) (*ObjectCatalog, error) {
	if src == nil {
		return nil, fmt.Errorf("%s: no graphics archive", ObjectRegistry)
	}
	stream, err := src.ReadFile(ObjectRegistry)
	if err != nil {
		return nil, err
	}
	r, err := reg.Parse(stream)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", ObjectRegistry, err)
	}
	classes, err := data.LoadObjectClasses(r)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", ObjectRegistry, err)
	}
	return &ObjectCatalog{classes: classes}, nil
}

// ByCode is the class a placement byte names.
func (c *ObjectCatalog) ByCode(b byte) (*data.ObjectClass, bool) { return c.classes.ByCode(b) }

// Name is the DescText of the class a placement byte names, empty when the
// byte names none or the class has no text.
func (c *ObjectCatalog) Name(b byte) string {
	if class, ok := c.ByCode(b); ok {
		return class.DescText
	}
	return ""
}
