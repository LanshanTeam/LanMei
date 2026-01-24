package dao

import "time"

func NextEmbeddingID() uint64 {
	if SFNode != nil {
		return uint64(SFNode.Generate().Int64())
	}
	return uint64(time.Now().UnixNano())
}
