# eventpix

Self-hostable event picture collection software, with bring your own storage.

## Features
- hosted / self-hostable
- bring your own storage - even on hosted service, events are configured to store photos and thumbnails straight in your storage. No identifiable media is stored in the apps database, just IDs
- retain metadata - photos uploaded maintain original EXIF metadata including date/time and location
- QR codes - create custom coloured QR codes for events in the app - including pre-adding events auth password into the QR code
- Optionally password protect events
- Unlimited file uploads (depending on how much storage you have!)
- Custom slug for event (i.e. your URL can be eventpix.com/my-awesome-event)
- If selfhosting, run in single event mode to make the landing page your configured "live" event (so can set photos.example.com to open straight into your guests gallery)

  ## Running
  - Create a copy of the `config.yaml` and fill in appropriate fields:
    - Create an encryption key for the db with `openssl rand -base64 32`
    - Oauth providers optional if you want to support cloud storage e.g. google drive, if ommitted, that provider will be exluded from the app when configuring an event
  - NATS for pub/sub can be configured to run in process, or can run seperately as a container in the compose stack
  - Thumbnailer service listens for events from NATS to generate thumbnails, so scan scale this out horizontally with multiple replicas in docker 