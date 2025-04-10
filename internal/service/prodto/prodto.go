package prodto

import (
	"github.com/jj-style/eventpix/internal/data/db"
	picturev1 "github.com/jj-style/eventpix/internal/gen/picture/v1"
	"github.com/samber/lo"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func Event(e *db.Event, withFileInfos bool) *picturev1.Event {
	ret := &picturev1.Event{
		Id:     uint64(e.ID),
		Name:   e.Name,
		Live:   e.Live,
		Active: e.Active,
	}
	if withFileInfos {
		ret.FileInfos = &picturev1.FileInfosValue{
			Value: lo.Map(e.FileInfos, func(item db.FileInfo, _ int) *picturev1.FileInfo { return FileInfo(&item) }),
		}
	}
	if e.Password != nil {
		ret.Password = wrapperspb.String(e.Password.Raw.(string))
	}
	return ret
}

func FileInfo(fi *db.FileInfo) *picturev1.FileInfo {
	return &picturev1.FileInfo{
		Id:      fi.ID,
		Name:    fi.Name,
		Video:   fi.Video,
		EventId: uint64(fi.EventID),
	}
}

func CreateEventResponse(id uint) *picturev1.CreateEventResponse {
	return &picturev1.CreateEventResponse{
		Id: uint64(id),
	}
}

func Thumbnail(ti *db.ThumbnailInfo) *picturev1.Thumbnail {
	return &picturev1.Thumbnail{
		Id:   ti.ID,
		Name: ti.Name,
		FileInfo: &picturev1.FileInfo{
			Id:      ti.FileInfoID,
			Name:    ti.FileInfo.Name,
			Video:   ti.FileInfo.Video,
			EventId: uint64(ti.FileInfo.EventID),
		},
		EventId: uint64(ti.EventID),
	}
}
