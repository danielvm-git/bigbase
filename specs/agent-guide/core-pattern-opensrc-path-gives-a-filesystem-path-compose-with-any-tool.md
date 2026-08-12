# Core pattern: opensrc path gives a filesystem path, compose with any tool

source: AGENTS.md
references: [AGENTS.md]

# Core pattern: opensrc path gives a filesystem path, compose with any tool
rg "pattern"     $(opensrc path zod)           # search source
cat               $(opensrc path zod)/src/types.ts  # read a file
find              $(opensrc path zod) -name "*.test.ts"  # explore
ls                $(opensrc path zod)/src/      # list directory
