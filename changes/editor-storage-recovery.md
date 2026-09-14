### Fixed

- Scene Browser caches analysis and thumbnails in browser storage, so Cloud projects no longer require a picked local folder.
- Reversed, processed, and nested-sequence audio previews handle unsupported media without an uncaught decode failure and clears the previous source's audio while a new source loads.
- Local file-handle storage reopens closed IndexedDB connections before starting a transaction.
