import { useEffect, useRef, useState } from "react";
import { useClient } from "../../services/useClient";
import { PictureService } from "../../gen/picture/v1/picture_connect";
import {
  GetThumbnailsRequest,
  GetThumbnailsResponse,
  Thumbnail,
  UploadRequest,

} from "../../gen/picture/v1/picture_pb";
import { Fab } from "react-tiny-fab";
import "react-tiny-fab/dist/styles.css";
import { Button, Container, Navbar, Spinner } from "react-bootstrap";
import Form from "react-bootstrap/Form";
import { Result } from "../../types/result";
import { Check, X } from "react-bootstrap-icons";
import InfiniteScroll from "react-infinite-scroll-component";
import ThumbnailImage from "../thumbnail/thumbnail";


const FilesUpload: React.FC = () => {
  // file upload state
  const [selectedFiles, setSelectedFiles] = useState<FileList | null>(null);
  const [uploading, setUploading] = useState<boolean>(false);
  const [previousUpload, setPreviousUpload] = useState<Result<boolean> | null>(
    null
  );
  const [thumbnails, setThumbnails] = useState<Array<Thumbnail>>([]);
  const inputFile = useRef<HTMLInputElement | null>(null);

  // scrolling state
  const [hasMore, setHasMore] = useState<boolean>(true);
  const [offset, setOffset] = useState<bigint>(BigInt(0));
  const limit = BigInt(30);

  const client = useClient(PictureService);

  const fetchData = () => {
    console.log("fetching");
    client.getThumbnails(new GetThumbnailsRequest({eventId: BigInt(1), offset: offset, limit: limit }))
    .then((response) => {
      const res = response as GetThumbnailsResponse;
      setThumbnails(curr => curr.concat(res.thumbnails));
      setHasMore(res.thumbnails.length === Number(limit));
      setOffset(curr => curr+BigInt(res.thumbnails.length));
    });
  }
  
  const refresh = () => {
    console.log("refreshing");
    fetchData();
  }
  
  useEffect(() => {
    console.log("initial load");
    fetchData();
  }, [])


  const selectFiles = (event: React.ChangeEvent<HTMLInputElement>) => {
    setSelectedFiles(event.target.files);
  };

  const upload = async (idx: number, file: File) => {
    var buffer = await file.arrayBuffer();
    let byteArray = new Uint8Array(buffer);

    return client
      .upload(new UploadRequest({ file: { name: file.name, data: byteArray } }))
      .then(() => {})
      .catch((err: any) => {
        let msg = file.name + ": Failed!";
        console.error(msg, err);
        throw new Error(err);
      });
  };

  const uploadFiles = () => {
    if (selectedFiles != null) {
      setUploading(true);
      const files = Array.from(selectedFiles);

      const uploadPromises = files.map((file, i) => upload(i, file));

      Promise.all(uploadPromises)
        .then(() => {
          setSelectedFiles(null);
          setPreviousUpload({ ok: true, data: true });
        })
        .catch((err) => {
          console.error("error uploading all files", err);
          setPreviousUpload({ error: err, ok: false });
        })
        .finally(() => {
          setUploading(false);
        });
    }
  };

  console.log(previousUpload);

  return (
    <Container>
      <Navbar className="bg-body-tertiary justify-content-between">
        <Navbar.Brand>Gallery</Navbar.Brand>
        <Form
          onSubmit={(e) => {
            e.preventDefault();
            uploadFiles();
          }}
        >
          <Form.Control
            type="file"
            multiple
            ref={inputFile}
            onChange={selectFiles}
            hidden
          />

          <Button
            variant={
              previousUpload === null
                ? "primary"
                : previousUpload.ok
                ? "success"
                : "danger"
            }
            type="submit"
            disabled={!selectedFiles}
            hidden={!previousUpload && !selectedFiles}
          >
            {uploading ? (
              <Spinner
                as="span"
                animation="border"
                size="sm"
                role="status"
                aria-hidden="true"
              />
            ) : selectedFiles ? (
              <span>Upload {selectedFiles?.length} files</span>
            ) : previousUpload?.ok ? (
              <Check />
            ) : (
              <X />
            )}
          </Button>
        </Form>
      </Navbar>
      <Fab
        event="click"
        onClick={() => {
          inputFile?.current?.click();
        }}
        style={{ bottom: 0, right: 0 }}
        mainButtonStyles={{ backgroundColor: "red" }}
        icon="+️"
      ></Fab>
      <InfiniteScroll
        className="d-flex flex-wrap"
        dataLength={thumbnails.length} //This is important field to render the next data
        next={fetchData}
        hasMore={hasMore}
        loader={<h4>Loading...</h4>}
        endMessage={
          <p style={{ textAlign: "center" }}>
            <b>Yay! You have seen it all</b>
          </p>
        }
        // below props only if you need pull down functionality
        refreshFunction={refresh}
        pullDownToRefresh
        pullDownToRefreshThreshold={50}
        pullDownToRefreshContent={
          <h3 style={{ textAlign: "center" }}>&#8595; Pull down to refresh</h3>
        }
        releaseToRefreshContent={
          <h3 style={{ textAlign: "center" }}>&#8593; Release to refresh</h3>
        }
      >
        {thumbnails.map((item, idx) => <ThumbnailImage key={idx} thumbnail={item} />)}
      </InfiniteScroll>
    </Container>
  );
};

export default FilesUpload;
