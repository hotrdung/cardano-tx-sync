// internal/transformer/transformer.go
package transformer

import (
	"cardano-tx-sync/internal/model"
	"cardano-tx-sync/internal/pb"
	"cardano-tx-sync/internal/utils"
	"encoding/json"
	"fmt"

	"github.com/SundaeSwap-finance/ogmigo/ouroboros/chainsync"
	"github.com/SundaeSwap-finance/ogmigo/ouroboros/shared"
	"github.com/gogo/protobuf/types"
)

// ToCardanoTransaction converts a model.TxnMessage to a pb.CardanoTransaction.
func ToCardanoTransaction(txMsg model.TxnMessage) (*pb.CardanoTransaction, error) {
	tx := txMsg.Tx
	block := txMsg.Block

	// Note:
	// with Sundae Ogmigo, the Redeemers is a JSON array of objects different from our Ogmigo version
	// to use it, we have to convert RPC definition to []*Struct instead of Struct
	// Example:
	// [ {"validator":{"index":0,"purpose":"spend"},"redeemer":"1a002dc6c0","executionUnits":{"memory":2894216,"cpu":983272159}},
	//   {"validator":{"index":2,"purpose":"spend"},"redeemer":"d87980","executionUnits":{"memory":23776,"cpu":7740640}} ]
	//
	// Refer to test data:
	// https://github.com/SundaeSwap-finance/ogmigo/blob/6211ee30eaa35d0955673a117cb0f3bf834044c0/ouroboros/chainsync/types_test.go#L878

	txBody, err := transformBody(tx)
	if err != nil {
		// have logged inside transformBody()
		// fmt.Printf("[Transformer] cannot parse body to proto message, datums: %v, error: %v\n", tx.Datums, err)
		return nil, err
	}
	datums := make(map[string][]byte)
	for key, value := range tx.Datums {
		datums[key] = []byte(value)
	}
	metadata, err := transformMetadata(tx.Metadata)
	if err != nil {
		fmt.Printf("[Transformer] cannot parse metadata to proto message, metadata: %v, error: %v\n", tx.Metadata, err)
		return nil, err
	}
	redeemers, err := transformRedeemers(tx.Redeemers)
	if err != nil {
		fmt.Printf("[Transformer] cannot parse redeemers to proto message, redeemers: %v, error: %v\n", tx.Redeemers, err)
		return nil, err
	}
	certs, err := transformCertificates(tx.Certificates)
	if err != nil {
		fmt.Printf("[Transformer] cannot parse certificates to proto message, certs: %v, error: %v\n", tx.Certificates, err)
		return nil, err
	}
	proposals, err := transformProposals(tx.Proposals)
	if err != nil {
		fmt.Printf("[Transformer] cannot parse proposals to proto message, proposals: %v, error: %v\n", tx.Proposals, err)
		return nil, err
	}
	votes, err := transformVotes(tx.Votes)
	if err != nil {
		fmt.Printf("[Transformer] cannot parse votes to proto message, votes: %v, error: %v\n", tx.Votes, err)
		return nil, err
	}
	signatures := make(map[string]string)
	for _, signature := range tx.Signatories {
		signatures[signature.Key] = signature.Signature
	}

	cardanoTx := &pb.CardanoTransaction{
		TransactionID: tx.ID,
		HeaderHash:    block.Hash,
		Slot:          block.Slot,
		Cbor:          tx.CBOR,
		Body:          txBody,
		Datums:        datums,
		Redeemers:     redeemers,
		Metadata:      metadata,
		Certificates:  certs,
		Proposals:     proposals,
		Votes:         votes,
		Signatures:    signatures,
	}

	return cardanoTx, nil
}

func transformBody(tx chainsync.Tx) (*pb.TxBody, error) {
	outputs, err := transformOutputs(tx.Outputs)
	if err != nil {
		fmt.Printf("[Transformer] cannot parse outputs to proto message, outputs: %v, error: %v\n", tx.Outputs, err)
		return nil, err
	}

	return &pb.TxBody{
		Inputs:                transformInputs(tx.Inputs),
		Outputs:               outputs,
		Mint:                  transformMint(tx.Mint),
		ReferenceInputs:       transformReferences(tx.References),
		ValidityIntervalStart: int64(tx.ValidityInterval.InvalidAfter),
		ValidityIntervalEnd:   int64(tx.ValidityInterval.InvalidBefore),
		Withdrawals:           transformWithdrawals(tx.Withdrawals),
	}, nil
}

func transformInputs(inputs []chainsync.TxIn) []*pb.TxInput {
	pbInputs := make([]*pb.TxInput, len(inputs))
	for i, in := range inputs {
		pbInputs[i] = &pb.TxInput{
			TransactionID: string(in.TxID()),
			Index:         int32(in.Index),
		}
	}
	return pbInputs
}

func transformOutputs(outputs []chainsync.TxOut) ([]*pb.TxOutput, error) {
	pbOutputs := make([]*pb.TxOutput, len(outputs))
	for i, out := range outputs {
		script, err := utils.ConvertToProtobufStruct(out.Script)
		if err != nil {
			return nil, fmt.Errorf("failed to transform output script: %w", err)
		}
		pbOutputs[i] = &pb.TxOutput{
			Address:   out.Address,
			Datum:     out.Datum,
			DatumHash: out.DatumHash,
			Value:     transformOutputValue(out.Value),
			Script:    script,
		}
	}
	return pbOutputs, nil
}

func transformOutputValue(value shared.Value) *pb.OutputValue {
	assets := make(map[string]int64)
	lovelaceAmount := int64(0)

	for policyId, items := range value {
		for assetName, qty := range items {
			if policyId == shared.AdaPolicy && assetName == shared.AdaAsset {
				// Coins (Lovelace)
				lovelaceAmount = qty.Int64()
			} else {
				assetStr := utils.GetAsset(policyId, assetName)
				assets[assetStr] = qty.Int64()
			}
		}
	}

	return &pb.OutputValue{
		Coins:  lovelaceAmount,
		Assets: assets,
	}
}

func transformMint(value shared.Value) *pb.MintValue {
	assets := make(map[string]int64)
	lovelaceAmount := int64(0)

	for policyId, items := range value {
		for assetName, qty := range items {
			if policyId == shared.AdaPolicy && assetName == shared.AdaAsset {
				// Coins (Lovelace)
				lovelaceAmount = qty.Int64()
			} else {
				assetStr := utils.GetAsset(policyId, assetName)
				assets[assetStr] = qty.Int64()
			}
		}
	}

	return &pb.MintValue{
		Coins:  lovelaceAmount,
		Assets: assets,
	}
}

func transformReferences(refs []chainsync.TxIn) []*pb.Reference {
	// "references": [
	// 	{
	// 		"transaction": {
	// 			"id": "22d10d66cdcc3ea3deaefa5b8fa2c3fe5fdfcbd4cf308a65d003ad2a93ee3179"
	// 		},
	// 		"index": 5
	// 	},
	// 	{
	// 		"transaction": {
	// 			"id": "ac7891786c12ad97f3a673eda87a07bbbaaec6c196161e29193e23b06b59294a"
	// 		},
	// 		"index": 0
	// 	},
	// 	{
	// 		"transaction": {
	// 			"id": "e17110fd84f7517d6fddca91ab151aefefa7f3c022a7499be040378358f5d94b"
	// 		},
	// 		"index": 1
	// 	}
	// ],

	if len(refs) == 0 {
		return nil
	}
	pbRefs := make([]*pb.Reference, len(refs))
	for i, ref := range refs {
		pbRefs[i] = &pb.Reference{
			TransactionID: ref.Transaction.ID,
			Index:         int32(ref.Index),
		}
	}
	return pbRefs
}

func transformWithdrawals(withdrawals map[string]shared.Value) map[string]map[string]int64 {
	// "withdrawals": {
	// 	"stake17xd7s38syqung8dqh2eu9erwcejda2y0njle0tt880ljunq6glahd": {
	// 		"ada": {
	// 			"lovelace": 893298
	// 		}
	// 	},
	// 	"stake1uxjy7wsp5ct2kjcpv7sec9mv6zm24mgyu4ls0rlj9rlp0wsvwx7xg": {
	// 		"ada": {
	// 			"lovelace": 367880
	// 		}
	// 	}
	// },

	if len(withdrawals) == 0 {
		return nil
	}
	results := make(map[string]map[string]int64) // { "$stakeAddr": {"$policyId.$assetName": qty} }
	for r, value := range withdrawals {
		results[r] = transformWithdrawalValue(value)
	}
	return results
}

func transformWithdrawalValue(value shared.Value) map[string]int64 {
	assets := make(map[string]int64)
	for policyId, items := range value {
		for assetName, qty := range items {
			assetStr := utils.GetAsset(policyId, assetName)
			assets[assetStr] = qty.Int64()
		}
	}
	return assets
}

func transformRedeemers(redeemersJSON json.RawMessage) (*types.Struct, error) {
	// "redeemers": [
	// 	{
	// 		"validator": {
	// 			"index": 4,
	// 			"purpose": "spend"
	// 		},
	// 		"redeemer": "a5239fd87d9f004023014273aeff9f00418b2143ee26a901ffffa1d87d9f43bafcf405ff425ccc029fd87c9f40ff446e943aa9d87e9f4375079e4318997905054197ffff23a3d87d9f43ca6d3e044233884206eb20ff0505a2024022019f43570e4322402324ff9f423456ff01d87c9fd87e9f01ff9f01ffff00",
	// 		"executionUnits": {
	// 			"memory": 5528516116957021378,
	// 			"cpu": 3267967087510563235
	// 		}
	// 	},
	// 	{
	// 		"validator": {
	// 			"index": 2,
	// 			"purpose": "mint"
	// 		},
	// 		"redeemer": "d87b9f219f232041de009f44d47dc270ffffa3d87a9f0521ffa2024127054404aa1521d87980a24040444097f3a504d87d9f425ca843eb4ac704445f938c0905ff9f03ffa021ff",
	// 		"executionUnits": {
	// 			"memory": 5484661661513700435,
	// 			"cpu": 3616344145952188136
	// 		}
	// 	},
	// 	{
	// 		"validator": {
	// 			"index": 2,
	// 			"purpose": "withdraw"
	// 		},
	// 		"redeemer": "d87c9f9f80a12442533fffa1d87d9f0201ffd8799f01ffff",
	// 		"executionUnits": {
	// 			"memory": 2494865883185442907,
	// 			"cpu": 4035456766489310695
	// 		}
	// 	}
	// ]

	if len(redeemersJSON) == 0 {
		return nil, nil
	}

	var redeemers []map[string]interface{}
	if err := json.Unmarshal(redeemersJSON, &redeemers); err != nil {
		// b.Logger.Error(err, "[Transformer] cannot parse redeemers", "redeemers", redeemersJSON)
		return nil, err
	}

	fields := make(map[string]*types.Value)

	for _, redeemer := range redeemers {
		validator := redeemer["validator"].(map[string]interface{})
		purpose := validator["purpose"].(string)
		index := validator["index"].(float64)
		redeemerValue := redeemer["redeemer"].(string)

		key := fmt.Sprintf("%s:%d", purpose, int(index))
		fields[key] = &types.Value{
			Kind: &types.Value_StringValue{
				StringValue: redeemerValue,
			},
		}
	}

	return &types.Struct{
		Fields: fields,
	}, nil
}

func transformMetadata(metadata json.RawMessage) (*types.Struct, error) {
	// "metadata": {
	// 	"hash": "5de7c98c16812894f26804d889e1223812f92362a6e6bebef61e0e3602220a2b",
	// 	"labels": {
	// 		"1": {
	// 			"json": -4
	// 		},
	// 		"6": {
	// 			"cbor": "40"
	// 		}
	// 	}
	// },

	if len(metadata) == 0 {
		return nil, nil
	}
	return utils.ConvertToProtobufStruct(metadata)
}

func transformCertificates(certs []json.RawMessage) ([]*types.Struct, error) {
	// "certificates": [
	// 	{
	// 		"type": "stakeDelegation",
	// 		"credential": "800fabbca2f56cb7a428a8066e3d8354ebc5bb882179924fa94dbad1",
	// 		"stakePool": {
	// 			"id": "pool128dhwtz47afh3v3afvdzpy07t0jl6yp3qws29v0lkryhzhkhj0n"
	// 		}
	// 	}
	// ],

	if len(certs) == 0 {
		return nil, nil
	}
	var result []*types.Struct
	for _, cert := range certs {
		certData, err := utils.ConvertToProtobufStruct(cert)
		if err != nil {
			// b.Logger.Error(err, "[Transformer] cannot parse certificate data to proto message", "cert", cert)
			return nil, err
		}
		result = append(result, certData)
	}
	return result, nil
}

func transformProposals(proposals json.RawMessage) ([]*types.Struct, error) {
	// "proposals": [
	// 	{
	// 		"action": {
	// 			"type": "treasuryWithdrawals",
	// 			"withdrawals": {
	// 				"64519ff082ace5007781306a885bf04a6dfc6df57fe486d3d98b7cb5": {
	// 					"ada": {
	// 						"lovelace": -659851
	// 					}
	// 				},
	// 				"7e0531fb0ec714b6080d437377c91a56d3c4351e32ca2b27dd1f0195": {
	// 					"ada": {
	// 						"lovelace": 168719
	// 					}
	// 				},
	// 				"9c444af5980051bfc60ef4a54068cae4ba543461c8bd37b0c174c9b7": {
	// 					"ada": {
	// 						"lovelace": -562643
	// 					}
	// 				}
	// 			}
	// 		}
	// 	}
	// ],

	if len(proposals) == 0 {
		return nil, nil
	}
	return utils.ConvertJsonArrayRawMessageToProtobufStruct(proposals)
}

func transformVotes(votes json.RawMessage) ([]*types.Struct, error) {
	if len(votes) == 0 {
		return nil, nil
	}
	return utils.ConvertJsonArrayRawMessageToProtobufStruct(votes)
}
