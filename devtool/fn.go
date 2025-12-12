package devtool

import (
	"crypto/sha256"
	"encoding/base64"
	reflect "reflect"

	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/ctx"
)

func generateLayersByPattern(restLayers []common.RESTLayer) map[string][]*common.RESTLayer {
	result := map[string][]*common.RESTLayer{}

	for _, layer := range restLayers {
		if _, ok := result[layer.Pattern]; !ok {
			result[layer.Pattern] = []*common.RESTLayer{}
		}

		result[layer.Pattern] = append(
			result[layer.Pattern],
			&layer,
		)
	}

	return result
}

func generateHandlerID(str string) string {
	encoded := base64.RawURLEncoding.EncodeToString([]byte(str))
	hash := sha256.Sum256([]byte(encoded))
	encoded = base64.RawURLEncoding.EncodeToString(hash[:])
	return encoded[:12]
}

// TODO:
// shouldn't handle context since ctx quite generic
// need to handle ws payload
func generateRequestPayload(pipe reflect.Type) (string, []*Schema) {
	pipeableTypes := map[string]reflect.Type{
		common.BODY_PIPEABLE:   reflect.TypeOf((*common.BodyPipeable)(nil)).Elem(),
		common.FORM_PIPEABLE:   reflect.TypeOf((*common.FormPipeable)(nil)).Elem(),
		common.QUERY_PIPEABLE:  reflect.TypeOf((*common.QueryPipeable)(nil)).Elem(),
		common.HEADER_PIPEABLE: reflect.TypeOf((*common.HeaderPipeable)(nil)).Elem(),
		common.PARAM_PIPEABLE:  reflect.TypeOf((*common.ParamPipeable)(nil)).Elem(),
		common.FILE_PIPEABLE:   reflect.TypeOf((*common.FilePipeable)(nil)).Elem(),
	}

	for pipeableKey, interfaceType := range pipeableTypes {
		if pipe.Implements(interfaceType) {
			return pipeableKey, GenerateSchema(pipe, ctx.TagBind)
		}
	}

	return "", nil
}
