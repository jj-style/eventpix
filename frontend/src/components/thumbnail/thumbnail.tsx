import { Image } from "react-bootstrap";
import { GetThumbnailRequest, GetThumbnailResponse, Thumbnail } from "../../gen/picture/v1/picture_pb";
import { useEffect, useState } from "react";
import { useClient } from "../../services/useClient";
import { PictureService } from "../../gen/picture/v1/picture_connect";

interface ThumbnailImageProps {
  thumbnail: Thumbnail;
}

// https://placehold.co/64png
const PLACEHOLDER = "iVBORw0KGgoAAAANSUhEUgAAAEAAAABACAYAAACqaXHeAAAACXBIWXMAAAsTAAALEwEAmpwYAAACzklEQVR4nOWW6U4iQRRGv/d/AxcEAQ3BYNwiLhGjUeJKFEVw4Vlq8t3YRlmcZAx9MjM/Tlhud9e5SxVoOBym/xnRAjSiBWhEC9CIFqARLUAjWoBGtACNaAEa0QI0ogVoRAvQiBagES1AI1qARrQAjWgBGtECNKIFaEQL0IgWoBEtQCNagEa0AI1oARrRAjSiBf7qAnQ6nXR6epqen5/HYvf39xF7eXn5seTb21tqt9vBpPjl5eXU2MwK0Ol00vz8fGo2m2OxwWCQlpeX09zcXHp6evpxAQ4ODmItJzoau7m5idjKykp+BRgMBqlUKqW1tbXozmi80WikarU6tQCPj4+p3+9/fO52u1Mn5fr6OhJstVpjMT+7WCzGWrkWoNFopHK5HOKjBbCopSw+rQB7e3sh7SLc3d2lpaWldH5+Pnad437WxsbGl4IZr1uv18Pl6OgovwK02+1IzF3xq+Wz0by9vY3vr66u4gyYVgDLb21tRRF8/6Tumu3t7S9rra6upl6vF7HDw8OYQk9jrgWo1+vRfY+xF/c2cBLZvnd3fd13BTDuvBNzEqPdNd4Sjrv7fu9p8zqbm5sf+96F9rW5FqBcLn8kac7OziLRk5OTeJ2Ekx1NPuu8u5xth8/XPDw8xL3eStl3LoanYGdnZ+I6LubMC7C+vp5qtVp6fX2NUbaUF/Z7J5aRbRV3afSc8PhnY++Yi3B8fDy2TQqFQtrf3/9y8Ppe/+x+Xmt3dzdVKpUo2swL0O12Y9TdQWPJbBQ/87sz4LvPGT4YFxYWIvHFxcWYlEnPy3ULDN/3p5O+uLiIzky7xuPrSfmTNTKcsKfpu2f5PPL/ktwKMPyHEC1AI1qARrQAjWgBGtECNKIFaEQL0IgWoBEtQCNagEa0AI1oARrRAjSiBWhEC9CIFqARLUAjWoBGtACNaAEa0QI0ogVoRAvQiBagES1AI1qA5heBXAq6xs8YZAAAAABJRU5ErkJggg==";

const ThumbnailImage = ({ thumbnail }: ThumbnailImageProps) => {
    const [imageData, setImageData] = useState<string>(PLACEHOLDER);
    const client = useClient(PictureService);

    useEffect(() => {
        client.getThumbnail(new GetThumbnailRequest({id: thumbnail.id}))
        .then(res => (res as GetThumbnailResponse).data)
        .then(data => setImageData(data));
    }, [])

    return <div className="my-1" style={{flex: "0 0 33%"}}> <Image
        src={`data:image/png;base64,${imageData}`}
        alt={`Thumbnail for ${thumbnail.fileInfo?.name}`} 
        rounded 
    /></div>
};

export default ThumbnailImage;
