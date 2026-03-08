#!/usr/bin/env python3
"""Convert results.txt to results.md with clickable GitHub links."""

import re


def convert_to_markdown(input_path, output_path):
    with open(input_path, 'r') as f:
        lines = f.readlines()

    out = []

    i = 0
    while i < len(lines):
        line = lines[i].rstrip('\n')
        stripped = line.strip()

        # Skip pure separator lines
        if re.match(r'^[=]{10,}$', stripped) or re.match(r'^[-]{10,}$', stripped):
            i += 1
            continue

        # Main title (first non-empty line before any section)
        if i == 0:
            out.append(f'# {stripped}\n')
            i += 1
            continue

        # Section header: a non-empty all-caps line followed by ----
        if (stripped and i + 1 < len(lines) and
                re.match(r'^[-]{10,}$', lines[i + 1].strip())):
            out.append(f'\n## {stripped}\n')
            i += 2  # skip dashes line
            continue

        # Repository URL line -> clickable link
        m = re.match(r'^(\s*)Repository:\s+(https?://\S+)\s*$', line)
        if m:
            url = m.group(2).rstrip('/')
            parts = url.split('/')
            display = '/'.join(parts[-2:]) if len(parts) >= 2 else url
            out.append(f'\n**Repository:** [{display}]({url})\n')
            i += 1
            continue

        # Server names
        m = re.match(r'^\s*Server names:\s+(.+)$', line)
        if m:
            out.append(f'**Server names:** {m.group(1).strip()}\n')
            i += 1
            continue

        # Sources
        m = re.match(r'^\s*Sources:\s+(.+)$', line)
        if m:
            out.append(f'**Sources:** {m.group(1).strip()}\n')
            i += 1
            continue

        # Subfolder
        m = re.match(r'^\s*Subfolder:\s+(.+)$', line)
        if m:
            out.append(f'**Subfolder:** {m.group(1).strip()}\n')
            i += 1
            continue

        # Classification
        m = re.match(r'^\s*Classification:\s+(.+)$', line)
        if m:
            out.append(f'**Classification:** `{m.group(1).strip()}`\n')
            i += 1
            continue

        # Evidence header
        m = re.match(r'^\s*Evidence \((\d+) matches?\):$', line)
        if m:
            out.append(f'**Evidence** ({m.group(1)} matches):\n')
            i += 1
            # Collect all evidence lines until blank line or next repo entry
            while i < len(lines):
                eline = lines[i].rstrip('\n')
                estripped = eline.strip()

                # Stop at blank line that precedes a new repo or section
                if estripped == '':
                    # Look ahead
                    j = i + 1
                    while j < len(lines) and lines[j].strip() == '':
                        j += 1
                    if j >= len(lines) or re.match(r'^\s*Repository:', lines[j]) or \
                       (lines[j].strip() and j + 1 < len(lines) and
                        re.match(r'^[-]{10,}$', lines[j + 1].strip())):
                        break
                    out.append('\n')
                    i += 1
                    continue

                # File path line (ends with colon)
                fm = re.match(r'^(\s{4,})(\S[^:]+\.(ts|js|py|go|java|rs|sh|txt|md|toml|json|yaml|yml)):$', eline)
                if fm:
                    out.append(f'- `{fm.group(2)}`\n')
                    i += 1
                    continue

                # "... and N more matches" line
                mm = re.match(r'^\s*(\.\.\. and \d+ more matches?)$', eline)
                if mm:
                    out.append(f'  - *{mm.group(1)}*\n')
                    i += 1
                    continue

                # Code snippet line (deeply indented)
                sm = re.match(r'^(\s{6,})(.+)$', eline)
                if sm:
                    out.append(f'  - `{sm.group(2).strip()}`\n')
                    i += 1
                    continue

                # Anything else - stop collecting evidence
                break
            continue

        # Summary stats: lines like "  Total unique repositories: 2427"
        m = re.match(r'^(\s+)(Total unique repositories|Successfully analyzed|Not found [^:]+|Other errors|Repos implementing BOTH[^:]+):\s+(\d+)\s*$', line)
        if m:
            out.append(f'{m.group(1)}{m.group(2)}: **{m.group(3)}**\n')
            i += 1
            continue

        # Convert any remaining GitHub URLs in text to links
        converted = re.sub(
            r'(https?://github\.com/[\w.\-]+/[\w.\-]+)',
            lambda mo: f'[{"/".join(mo.group(1).rstrip("/").split("/")[-2:])}]({mo.group(1)})',
            line
        )
        out.append(converted + '\n')
        i += 1

    with open(output_path, 'w') as f:
        f.writelines(out)

    print(f"Written {output_path}")


if __name__ == '__main__':
    convert_to_markdown(
        '/home/user/research/mcpservers/results.txt',
        '/home/user/research/mcpservers/results.md'
    )
