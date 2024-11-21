package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"connectrpc.com/connect"
	picturev1 "github.com/jj-style/eventpix/backend/gen/picture/v1"
	"github.com/jj-style/eventpix/backend/gen/picture/v1/picturev1connect"
	"github.com/samber/lo"
)

var (
	ctx    = context.Background()
	client = picturev1connect.NewPictureServiceClient(http.DefaultClient, "http://localhost:8080/")
)

func createEvent() *picturev1.CreateEventResponse {
	got := lo.Must(client.CreateEvent(ctx, connect.NewRequest(&picturev1.CreateEventRequest{
		Name: "my event",
		Live: false,
		Storage: &picturev1.CreateEventRequest_Filesystem{
			Filesystem: &picturev1.Filesystem{
				Directory: "/ramdisk",
			},
		},
	})))
	fmt.Printf("==> created %+v\n", got.Msg)
	return got.Msg
}

func getEvents() []*picturev1.Event {
	got := lo.Must(client.GetEvents(ctx, connect.NewRequest(&picturev1.GetEventsRequest{})))
	for _, e := range got.Msg.GetEvents() {
		fmt.Printf("==> event: %+v\n", e)
	}
	return got.Msg.GetEvents()
}

func getEvent(id uint) *picturev1.Event {
	got := lo.Must(client.GetEvent(ctx, connect.NewRequest(&picturev1.GetEventRequest{Id: uint64(id)})))
	fmt.Printf("==> event: %+v\n", got.Msg.GetEvent())
	return got.Msg.GetEvent()
}

func getThumbnails(id uint) {
	got := lo.Must(client.GetThumbnails(ctx, connect.NewRequest(&picturev1.GetThumbnailsRequest{
		EventId: uint64(id),
		Limit:   -1,
		Offset:  0,
	})))
	for _, tb := range got.Msg.GetThumbnails() {
		fmt.Printf("===> got thumbnail: %+v\n", tb)
	}
}

func upload(evt uint64) {
	requests := []*picturev1.UploadRequest{
		{File: &picturev1.File{Name: "file1.png", Data: lo.Must(os.ReadFile("/home/jj/Pictures/wallpaper.png"))}, EventId: evt},
		{File: &picturev1.File{Name: "file2.png", Data: lo.Must(os.ReadFile("/home/jj/Pictures/wallpaper.png"))}, EventId: evt},
	}
	for _, req := range requests {
		resp, err := client.Upload(ctx, connect.NewRequest(req))
		if err != nil {
			log.Printf("==> err receiving: %v", err)
		}
		fmt.Printf("==> resp: %+v\n", resp.Msg)
	}
}

func main() {
	// createEvent()
	// upload(1)
	// getEvents()
	// getEvent(1)
	getThumbnails(1)
}
