### Fixed

- Deleting or deselecting a text item while its on-canvas editor is focused no longer throws `Cannot read properties of undefined (reading 'text')`. The removal blur now ends the editing session without committing, since there is nothing to commit to.
