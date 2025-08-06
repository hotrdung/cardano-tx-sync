package utils

import (
	"strings"
)

const (
	EmptyPolicyId     = ""
	EmptyAssetName    = ""
	AssetNameSplitter = "."
	ZeroIndex         = 0
)

func GetAsset(policyId string, assetName string) string {
	if assetName == EmptyAssetName {
		return policyId
	} else {
		return policyId + AssetNameSplitter + assetName
	}
}

func ParseAsset(asset string) (string, string) {
	if dotIndex := strings.Index(asset, AssetNameSplitter); dotIndex > ZeroIndex {
		return asset[:dotIndex], asset[dotIndex+1:]
	} else {
		return asset, EmptyAssetName
	}
}
