# Releases

`VERSION` is the shared version for the backend, GUI, filenames, and releases.
Use semantic versions: `0.1.0-alpha.1`, `0.1.0-alpha.2`, then `0.1.0` when stable.
Tags must match VERSION exactly; for `0.1.0-alpha.5`, use `v0.1.0-alpha.5`.

Branch pushes and pull/merge requests build and test the application.
Release tags build, test, and package the AppImage and tarball before publishing.
GitHub manual runs also validate packaging without publishing a release.
GitLab manual web pipelines can validate packaging too.

Tag runs test the exact tagged source independently, even when the same commit
already passed on main. Packaging is no longer repeated on ordinary branch pushes.
Only tags publish releases.

## Publish

1. Update `VERSION` and `CHANGELOG.md`, then run the build and tests.
2. Commit the change and push a matching `v…` tag.
3. CI builds and tests the AppImage and bundled tarball, generates SHA-256 sums,
   and publishes the files plus corresponding source.

GitHub Actions publishes to GitHub Releases using the built-in token. GitLab CI
publishes permanent Generic Package files and a GitLab Release using `CI_JOB_TOKEN`.

The app links to GitHub Releases when a newer version is available. Users install
new releases manually; automatic installation is a future feature.

## Build packages locally

Build on Debian 13 x86_64 (the CI container is `golang:1.26-trixie`):

```sh
bash scripts/build.sh
bash scripts/package.sh
```

Artifacts go to ignored `dist/`. Packaging needs curl/network access, patchelf,
file, desktop-file-utils, Qt SVG and image format plugins. Official linuxdeploy,
its Qt plugin, appimagetool, and the AppImage runtime are downloaded into ignored `.tools/` and checked
against `packaging/tools.lock.json`. If an upstream continuous release changes,
review its official asset SHA-256 before updating the lock; never disable checking.
Move/remove `build/AppDir` before repeating a package build.

The tarball and AppImage share an AppDir containing the decoder, GUI, Qt libraries,
QML modules, image plugins, and Wayland/layer-shell plugins. The bundled dumpcap helper and its libraries are extracted to a user-owned cache
so Polkit can authorize capture outside an AppImage FUSE mount. A working system
dumpcap is reused when already permitted.
System glibc, graphics drivers, the desktop compositor, and Polkit remain OS
requirements. No package-manager installation is performed on launch.
CI's Debian 13 baseline requires glibc 2.41+; building on a newer host can raise
that minimum. Current releases support x86_64 only.

Release files include `SHA256SUMS` for manual download verification.
