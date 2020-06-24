package test

import (
	"fmt"
	"testing"
	"time"

	"github.com/QuarkChain/goquarkchain/consensus/qkchash"
	"github.com/QuarkChain/goquarkchain/consensus/qkchash/native"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
)

const (
	cacheEntryCnt   = 1024 * 64
	cacheAheadRound = 64 // 64*30000
)

var (
	EpochLength = uint64(100) //blocks pre epoch
	caches      []*cache
)

// cache is the union type of cache for qkchash algo.
// Note in Go impl, `nativeCache` will be empty.
type cache struct {
	nativeCache native.Cache
	ls          []uint64
	seed        []byte
}

func getCacheFromHeight(block uint64) *cache {
	epoch := int(block / EpochLength)
	lenCaches := len(caches)
	if epoch < lenCaches {
		return caches[epoch]
	}
	needAddCnt := epoch - lenCaches + cacheAheadRound
	seed := caches[len(caches)-1].seed
	for i := 0; i < needAddCnt; i++ {
		seed = crypto.Keccak256(seed)
		caches = append(caches, generateCache(cacheEntryCnt, seed))
	}
	return caches[epoch]
}

// generateCache generates cache for qkchash. Will also generate underlying cache
func generateCache(cnt int, seed []byte) *cache {
	ls := qkchash.Generatels(cnt, seed)
	return &cache{native.NewCache(ls), ls, seed}
}

func TestQkcHashXCompare(t *testing.T) {
	ux := []bool{false, true}
	firstCache := generateCache(cacheEntryCnt, common.Hash{}.Bytes())
	caches = make([]*cache, 0)
	caches = append(caches, firstCache)
	seed := make([]byte, 40)
	tm := time.Now()
	round := uint64(500)
	for i := uint64(0); i < round; i++ {
		cache := getCacheFromHeight(i * 10)
		seed = crypto.Keccak512(seed)
		for _, usex := range ux {
			if err := CompareQkcHashBetweenGoAndNative(seed, cache, usex); err != nil {
				assert.NoError(t, err, fmt.Sprintf("compare error for round %d; \r\n\tseed: %v;"+
					"\r\n\tcache: %v; \r\n\tusex: %v;\r\n\terror: %v", i, seed, cache.ls, usex, err.Error()))
				return
			}
		}
	}
	fmt.Printf("test done, using %v seconds for %d round", time.Now().Sub(tm).Seconds(), round)
}

func CompareQkcHashBetweenGoAndNative(seed []byte, cache *cache, useX bool) error {
	resultN, errN := native.HashWithRotationStats(cache.nativeCache, seed, useX)
	resultG, errG := qkchash.HashWithRotationStats(cache.ls, seed, useX)
	if errN != nil {
		return fmt.Errorf("errN: %v", errN.Error())
	}
	if errG != nil {
		return fmt.Errorf("errG: %v", errG.Error())
	}
	for i := 0; i < 4; i++ {
		if resultG[i] != resultN[i] {
			return fmt.Errorf("diff : %v vs %v", resultN, resultG)
		}
	}
	return nil
}
