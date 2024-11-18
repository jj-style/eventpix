import { useEffect, useRef, useState } from "react";
import { useClient } from "../../services/useClient";
import { PictureService } from "../../gen/picture/v1/picture_connect";
import {
  UploadRequest,
  ListRequest,
  ListResponse,
} from "../../gen/picture/v1/picture_pb";
import { File as Filepb } from "../../gen/picture/v1/picture_pb";
import { Fab } from "react-tiny-fab";
import "react-tiny-fab/dist/styles.css";
import { Button, Container, Navbar, Spinner } from "react-bootstrap";
import Form from "react-bootstrap/Form";
import { Result } from "../../types/result";
import { Check, X } from "react-bootstrap-icons";
import { Message } from "@bufbuild/protobuf";

const FilesUpload: React.FC = () => {
  const [selectedFiles, setSelectedFiles] = useState<FileList | null>(null);
  const [uploading, setUploading] = useState<boolean>(false);
  const [previousUpload, setPreviousUpload] = useState<Result<boolean> | null>(
    null
  );
  const [fileInfos, setFileInfos] = useState<Array<Filepb>>([]);
  const [refresh, setRefresh] = useState(0);
  const inputFile = useRef<HTMLInputElement | null>(null);
  const client = useClient(PictureService);

  useEffect(() => {
    async function listGallery(){
      const res = await client.listGallery(new ListRequest({}));
      console.log(res);
      setFileInfos((res as ListResponse).files);
    }
    listGallery();
  }, [refresh, client]);

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
          setRefresh((prev) => prev + 1);
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
      <div className="card">
        <div className="card-header">List of Files</div>
        <ul className="list-group list-group-flush">
          {fileInfos &&
            fileInfos.map((file, index) => (
              <li className="list-group-item" key={index}>
                <span>{file.name}</span>
              </li>
            ))}
        </ul>
      </div>
    </Container>
  );
};

export default FilesUpload;
