package utils

import (
	"encoding/json"

	"github.com/gogo/protobuf/types"
	"github.com/golang/protobuf/jsonpb"
)

func ConvertToProtobufStruct(myStruct interface{}) (*types.Struct, error) {
	jsonStr, err := json.Marshal(myStruct)
	if err != nil {
		return nil, err
	}

	structObj := &types.Struct{}
	err = jsonpb.UnmarshalString(string(jsonStr), structObj)
	if err != nil {
		return nil, err
	}

	return structObj, nil
}

func ConvertJsonArrayRawMessageToProtobufStruct(arrayRawMsg json.RawMessage) ([]*types.Struct, error) {
	structObjs := []*types.Struct{}

	if arrayRawMsg == nil {
		return structObjs, nil
	}

	// arrayRawMsg is a json.RawMessage that represents an array of objects
	// => convert it into an array of json.RawMessage with name `rawMsgs`
	var rawMsgs []json.RawMessage
	err := json.Unmarshal(arrayRawMsg, &rawMsgs)
	if err != nil {
		return nil, err
	}

	if len(rawMsgs) == 0 {
		return structObjs, nil
	}

	for _, rawMsg := range rawMsgs {
		structObj, err := ConvertToProtobufStruct(rawMsg)
		if err != nil {
			return nil, err
		}
		structObjs = append(structObjs, structObj)
	}

	return structObjs, nil
}
