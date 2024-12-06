import { Image } from "react-bootstrap";
import { Thumbnail } from "../../gen/picture/v1/picture_pb";

interface ThumbnailImageProps {
  thumbnail: Thumbnail;
}

const ThumbnailImage = ({ thumbnail }: ThumbnailImageProps) => {
    return <div className="my-1" style={{flex: "0 0 33%"}}> <Image
        src={`http://localhost:8080/storage/thumbnail/${thumbnail.id}`}
        alt={`Thumbnail for ${thumbnail.fileInfo?.name}`} 
        rounded 
    /></div>
};

export default ThumbnailImage;
