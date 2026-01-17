import re
import json


def extract_options(file_path):
    with open(file_path, "r") as f:
        content = f.read()

    categories = []
    current_category = None

    # Split by headings
    sections = re.split(r"\n(###? .*)\n", content)

    options_data = []

    for i in range(1, len(sections), 2):
        category_name = sections[i].strip("# ").strip()
        category_content = sections[i + 1]

        # Regex to find options like \--dir \=<DIR> or \--enable-rpc \[true|false\]
        # Pattern matches: \--[option-name] [optional values] [permalink symbol]
        # Then we look for description until next option or end of section
        option_pattern = r"\\--([a-zA-Z0-9-]+)\s*(\\\[[^\]]+\\\]|\\=<[^>]+>|\\=[a-zA-Z0-9|]+)?\s*\[¶\]"

        matches = list(re.finditer(option_pattern, category_content))

        for j, match in enumerate(matches):
            name = match.group(1)
            type_info = match.group(2)

            # Clean type info
            if type_info:
                type_info = type_info.replace("\\", "").replace("=", "").strip("[] ")
            else:
                type_info = (
                    "boolean"
                    if "true|false" in category_content[match.end() : match.end() + 100]
                    else "string"
                )

            # Get description
            start = match.end()
            end = (
                matches[j + 1].start()
                if j + 1 < len(matches)
                else len(category_content)
            )
            description_text = category_content[start:end].strip()

            # Extract default value
            default_match = re.search(r"Default:\s*`([^`]+)`", description_text)
            default_value = default_match.group(1) if default_match else None

            # Clean description (remove everything after "Default:")
            clean_description = re.split(r"Default:", description_text)[0].strip()
            # Remove permalink lines if any
            clean_description = re.sub(
                r"\(https://aria2\.github\.io/.*?\)", "", clean_description
            )

            options_data.append(
                {
                    "name": name,
                    "category": category_name,
                    "type": type_info,
                    "default": default_value,
                    "description": clean_description[:500] + "..."
                    if len(clean_description) > 500
                    else clean_description,
                }
            )

    return options_data


if __name__ == "__main__":
    options = extract_options("aria-docs.md")
    with open("src/lib/aria2-options.json", "w") as f:
        json.dump(options, f, indent=2)
    print(f"Extracted {len(options)} options.")
