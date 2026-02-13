import sys
import re

with open('cmd/tmux-intray/status.go', 'r') as f:
    lines = f.readlines()

# Insert storage import after cmd import
new_lines = []
for line in lines:
    new_lines.append(line)
    if '"github.com/cristianoliveira/tmux-intray/cmd"' in line:
        new_lines.append('\t"github.com/cristianoliveira/tmux-intray/internal/storage"\n')

# Now replace magic numbers
content = ''.join(new_lines)
content = re.sub(r'len\(fields\) <= 8', r'len(fields) <= storage.FieldLevel', content)
content = re.sub(r'fields\[8\]', r'fields[storage.FieldLevel]', content)
content = re.sub(r'len\(fields\) <= 5', r'len(fields) <= storage.FieldPane', content)
content = re.sub(r'fields\[3\]', r'fields[storage.FieldSession]', content)
content = re.sub(r'fields\[4\]', r'fields[storage.FieldWindow]', content)
content = re.sub(r'fields\[5\]', r'fields[storage.FieldPane]', content)

with open('cmd/tmux-intray/status.go', 'w') as f:
    f.write(content)
