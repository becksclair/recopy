#compdef recopy
_arguments \
  '(-h --help)'{-h,--help}'[show usage]' \
  '(-)'{-V,--version}'[print version]' \
  '--move[rename when possible]' \
  '--dry-run[plan only]' \
  '--mirror[delete extraneous destination data]' \
  '--profile[auto|lan|wan]:profile:(auto lan wan)' \
  '--no-reflink[disable reflink fast path]' \
  '--inplace[rsync --inplace]' \
  '--parallel[parallel workers]:parallel workers:' \
  '--transport[auto|rsync|btrfs]:transport:(auto rsync btrfs)' \
  '--prescan[metadata prescan]' \
  '--verify[re-verify data]' \
  '--one-file-system[stay on same filesystem]' \
  '--no-ui[disable Bubble Tea UI]' \
  'doctor[run system diagnostics]' \
  '*:paths:_files'
