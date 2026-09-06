package ai

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/llm"

	"gorm.io/gorm"
)

// packEmbedding 将 float32 向量打包为 little-endian blob。
func packEmbedding(v []float32) []byte {
	if len(v) == 0 {
		return nil
	}
	b := make([]byte, 4*len(v))
	for i, f := range v {
		binary.LittleEndian.PutUint32(b[i*4:], math.Float32bits(f))
	}
	return b
}

// unpackEmbedding 从 blob 解包 float32 向量。
func unpackEmbedding(b []byte) []float32 {
	if len(b) < 4 || len(b)%4 != 0 {
		return nil
	}
	n := len(b) / 4
	out := make([]float32, n)
	for i := range n {
		out[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))
	}
	return out
}

func float64To32(v []float64) []float32 {
	out := make([]float32, len(v))
	for i, x := range v {
		out[i] = float32(x)
	}
	return out
}

func cosineSimilarity(a, b []float32) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		fa, fb := float64(a[i]), float64(b[i])
		dot += fa * fb
		na += fa * fa
		nb += fb * fb
	}
	if na <= 0 || nb <= 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

// EmbedSyncReport 向量补齐结果。
type EmbedSyncReport struct {
	Updated int      `json:"updated"`
	Skipped int      `json:"skipped"`
	Failed  int      `json:"failed"`
	Errors  []string `json:"errors,omitempty"`
}

// SyncEmbeddings 为缺少 Embedding 的 KB chunk 补齐向量。
func (s *Service) SyncEmbeddings(ctx context.Context) (*EmbedSyncReport, error) {
	emb, err := s.queryEmbedder(ctx)
	if err != nil {
		return nil, err
	}
	var chunks []model.AiKbChunk
	_ = s.db.WithContext(ctx).
		Where("embedding IS NULL OR LENGTH(embedding) = 0").
		Order("id ASC").
		Limit(200).
		Find(&chunks).Error
	rep := &EmbedSyncReport{}
	for _, ch := range chunks {
		text := strings.TrimSpace(ch.HeadingPath + "\n" + ch.Content)
		if text == "" {
			rep.Skipped++
			continue
		}
		vecs, err := emb.Embed(ctx, []string{truncateStr(text, 6000)})
		if err != nil || len(vecs) == 0 {
			rep.Failed++
			if len(rep.Errors) < 20 {
				msg := "chunk " + fmt.Sprintf("%d", ch.ID) + ": "
				if err != nil {
					msg += err.Error()
				} else {
					msg += "empty embedding"
				}
				rep.Errors = append(rep.Errors, msg)
			}
			continue
		}
		blob := packEmbedding(float64To32(vecs[0]))
		if err := s.db.WithContext(ctx).Model(&model.AiKbChunk{}).
			Where("id = ?", ch.ID).
			Update("embedding", blob).Error; err != nil {
			rep.Failed++
			continue
		}
		rep.Updated++
	}
	return rep, nil
}

func (s *Service) queryEmbedder(ctx context.Context) (llm.Embedder, error) {
	cfg := s.resolved(ctx)
	timeout := cfg.TimeoutSec
	if timeout <= 0 {
		timeout = 60
	}
	// 优先 embedding 类型模型
	var row model.AiLLMModel
	err := s.db.WithContext(ctx).
		Where("enabled = ? AND model_type = ?", true, "embedding").
		Order("is_default DESC, id ASC").
		First(&row).Error
	if err == nil {
		cli, _, _, cerr := s.clientFromDBModel(&row, timeout)
		if cerr == nil {
			if emb, ok := cli.(llm.Embedder); ok {
				return emb, nil
			}
		}
	} else if err != gorm.ErrRecordNotFound {
		return nil, err
	}
	// 回退：当前默认 chat 客户端若支持 Embed
	cli, _, _, err := s.clientFor(ctx, &cfg, "")
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg("无可用 Embedder: " + err.Error())
	}
	emb, ok := cli.(llm.Embedder)
	if !ok {
		return nil, constants.ErrBadRequestWithMsg("当前模型不支持 /embeddings，请配置 embedding 类型模型")
	}
	return emb, nil
}

func (s *Service) queryEmbedding(ctx context.Context, text string) []float32 {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	emb, err := s.queryEmbedder(ctx)
	if err != nil {
		return nil
	}
	vecs, err := emb.Embed(ctx, []string{truncateStr(text, 4000)})
	if err != nil || len(vecs) == 0 {
		return nil
	}
	return float64To32(vecs[0])
}

// blendChunkScore 词法 + 语义融合：有两侧时 final = lexical*0.6 + semantic*0.4。
func blendChunkScore(lexical float64, hasSemantic bool, semantic float64) float64 {
	if !hasSemantic {
		return lexical
	}
	// 将 cosine [-1,1] 大致映射到与词法同量级： (sim+1)/2 * 放大
	sem := (semantic + 1) / 2 * math.Max(lexical, 1)
	return lexical*0.6 + sem*0.4
}

func applyHybridChunkScore(ch model.AiKbChunk, lexical float64, queryVec []float32) float64 {
	if len(queryVec) == 0 || len(ch.Embedding) == 0 {
		return lexical
	}
	docVec := unpackEmbedding(ch.Embedding)
	if len(docVec) == 0 {
		return lexical
	}
	sim := cosineSimilarity(queryVec, docVec)
	return blendChunkScore(lexical, true, sim)
}
