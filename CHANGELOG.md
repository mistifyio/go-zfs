# Change Log
All notable changes to this project will be documented in this file.
This project adheres to [Semantic Versioning](http://semver.org/).
This change log follows the advice of [Keep a CHANGELOG](https://github.com/olivierlacan/keep-a-changelog).


## [2.2.0] - (2016-05-24)
### Added
- Rename, Mount and Unmount methods.
- Parse more fields into Zpool type:
  - dedupratio
  - fragmentation
  - freeing
  - leaked
  - readonly
- Parse numbers in exact format.
- Add support Solaris (unreleased Solaris version).

### Changed
- Temporarily adjust TestDiff expected strings depending on ZFS version.


## [2.1.1] - 2015-05-29
### Fixed
- Ignoring first pool listed.
- Incorrect `zfs get` argument ordering.


## [2.1.0] - 2014-12-08
### Added
- Parse hardlink modification count returned from `zfs diff`.

### Fixed
- Continuing instead of erroring when rolling back a non-snapshot.


## [2.0.0] - 2014-12-02
### Added
- Flags for Destroy:
  - DESTROY_DEFAULT
  - DESTROY_RECURSIVE (`zfs destroy ... -r`)
  - DESTROY_RECURSIVE_CLONES (`zfs destroy ... -R`)
  - DESTROY_DEFER_DELETION (`zfs destroy ... -d`)
  - DESTROY_FORCE (`zfs destroy ... -f`)
…
- Diff method (`zfs diff`).
- LogicalUsed and Origin properties to Dataset.
- Type constants for Dataset.
- State constants for Zpool.
- Logger interface.
- Improve documentation.


## [1.0.0] - 2014-11-12


[2.2.0]: https://github.com/mistifyio/go-zfs/compare/v2.1.1...v2.2.0
[2.1.1]: https://github.com/mistifyio/go-zfs/compare/v2.1.0...v2.1.1
[2.1.0]: https://github.com/mistifyio/go-zfs/compare/v2.0.0...v2.1.0
[2.0.0]: https://github.com/mistifyio/go-zfs/compare/v1.0.0...v2.0.0
[2.0.0]: https://github.com/mistifyio/go-zfs/compare/v1.0.0...v2.0.0
[1.0.0]: https://github.com/mistifyio/go-zfs/compare/a642fad...v1.0.0
