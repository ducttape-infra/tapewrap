# gstow

A cross-platform reimplementation of [GNU stow](https://www.gnu.org/software/stow/) in Go.

Works on Linux and Windows using OS-native symlink APIs — no shell commands invoked.

## Usage

```
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

```sh
# Stow zsh and vim packages to the parent directory
cd ~/.dotfiles && gstow zsh vim

# Stow to an explicit target
gstow -t ~ zsh vim

# Unstow
gstow -D zsh

# Restow (useful after adding new files to a package)
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

Use `app go build` or `action build` if Go is not installed locally (runs inside a container):

```sh
app go build        # local arch
app go amd-build    # linux/amd64
app go arm-build    # linux/arm64

action build        # local arch
```

## Windows

Symlinks require **Developer Mode** or Administrator rights. Directory junctions
are used as a fallback for directory links when symlink creation is unavailable.

## License

MIT
