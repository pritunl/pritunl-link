package state

import (
	"fmt"
	"sync"
)

var (
	linkVersions    = map[string]int{}
	linkVersionLock = sync.Mutex{}
)

func GetLinkId(stateId string, i int, linkHash string) string {
	linkId := fmt.Sprintf("%s-%d-%s", stateId, i, linkHash)

	linkVersionLock.Lock()
	ver := linkVersions[linkId]
	linkVersionLock.Unlock()

	return fmt.Sprintf("%s_%08d", linkId, ver)
}

func IncLinkId(linkId string) string {
	linkId = linkId[:len(linkId)-9]

	linkVersionLock.Lock()
	ver := linkVersions[linkId]
	ver += 1
	linkVersions[linkId] = ver
	linkVersionLock.Unlock()

	return fmt.Sprintf("%s_%08d", linkId, ver)
}
