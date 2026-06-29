gstow
=====

[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/ducttape-infra/gstow)

A cross-platform reimplementation of [GNU stow](https://www.gnu.org/software/stow/) in Go,
to do resource and configuration management.

Works on Linux and Windows using OS-native symlink APIs — no shell commands invoked.


## Usage

```sh
gstow [-D] [-R] [-t TARGET] [-v] [-n] PACKAGE...
```

| Flag | Description |
|------|-------------|
| `-D` | Delete (unstow) packages |
| `-R` | Restow (unstow then stow again) |
| `-t DIR` | Target directory (default: parent of current directory) |
| `-v` | Verbose output |
| `-n` | Dry run — show what would happen without doing it |


## Examples

### Stow zsh and vim packages to the parent directory
```sh
cd ~/.dotfiles && gstow zsh vim
```

# Stow to an explicit target
```sh
gstow -t ~ zsh vim
```

### Unstow
```sh
gstow -D zsh
```

### Restow (useful after adding new files to a package)
```sh
gstow -R zsh vim
```


## Behaviour

Follows GNU stow **directory folding** semantics:

- Target path does not exist → create a symlink to the source (file or directory)
- Target path is a real directory → recurse into it
- Target path is an existing correct symlink → skip (already stowed)
- Target path is an existing symlink to something else → warn and skip
- Target path is a real file → warn and skip


## Building

Use `action build local` or `make` to build:

```sh
action build local
```


## Windows

Symlinks require **Developer Mode** or Administrator rights. Directory junctions
are used as a fallback for directory links when symlink creation is unavailable.

