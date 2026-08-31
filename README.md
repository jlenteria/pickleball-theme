# Pickleball Theme

This plugin provides a branded Pickleball captive portal and an admin settings page for Flarewifi.

## Development

The plugin uses Go, templ-generated views, and the Flarewifi SDK. The SDK version is recorded in `plugin.json` and dependencies are in `go.mod`.

Useful checks:

```sh
make test
make test-mono
make vet
make check
```

Build the deployable plugin with:

```sh
make plugin
```

## Configuration

Administrators can configure the venue name, logo, banner, booking URL, court rates, tutorials, affiliate recommendations, and the optional local rally game. Uploaded logos and content are stored through the plugin storage API so they survive plugin updates.

Uploads are limited to 2 MiB for images and 25 MiB for tutorial videos. Accepted image formats are PNG, JPG/JPEG, WEBP, GIF, and SVG. Accepted video formats are MP4, WebM, and MOV.
