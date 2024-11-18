package prodto

import (
	picturev1 "github.com/jj-style/eventpix/backend/gen/picture/v1"
	"github.com/jj-style/eventpix/backend/internal/data/db"
	"github.com/samber/lo"
)

func Event(e *db.Event, withFileInfos bool) *picturev1.Event {
	ret := &picturev1.Event{
		Id:   uint64(e.ID),
		Name: e.Name,
		Live: e.Live,
	}
	if withFileInfos {
		ret.FileInfos = &picturev1.FileInfosValue{
			Value: lo.Map(e.FileInfos, func(item db.FileInfo, _ int) *picturev1.FileInfo { return FileInfo(&item) }),
		}
	}
	return ret
}

func FileInfo(fi *db.FileInfo) *picturev1.FileInfo {
	return &picturev1.FileInfo{
		Id: fi.StoreID,
	}
}

func CreateEventResponse(id uint) *picturev1.CreateEventResponse {
	return &picturev1.CreateEventResponse{
		Id: uint64(id),
	}
}
