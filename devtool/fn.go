package devtool

import (
	"crypto/sha256"
	"encoding/base64"
	reflect "reflect"

	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/ctx"
)

func generateLayersByPattern(httpLayers []common.HTTPLayer) map[string][]*common.HTTPLayer {
	result := map[string][]*common.HTTPLayer{}

	for _, layer := range httpLayers {
		if _, ok := result[layer.Pattern]; !ok {
			result[layer.Pattern] = []*common.HTTPLayer{}
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
		common.BodyPipeableKey:   reflect.TypeOf((*common.BodyPipeable)(nil)).Elem(),
		common.FormPipeableKey:   reflect.TypeOf((*common.FormPipeable)(nil)).Elem(),
		common.QueryPipeableKey:  reflect.TypeOf((*common.QueryPipeable)(nil)).Elem(),
		common.HeaderPipeableKey: reflect.TypeOf((*common.HeaderPipeable)(nil)).Elem(),
		common.ParamPipeableKey:  reflect.TypeOf((*common.ParamPipeable)(nil)).Elem(),
		common.FilePipeableKey:   reflect.TypeOf((*common.FilePipeable)(nil)).Elem(),
	}

	for pipeableKey, interfaceType := range pipeableTypes {
		if pipe.Implements(interfaceType) {
			return pipeableKey, GenerateSchema(pipe, ctx.TagBind)
		}
	}

	return "", nil
}
