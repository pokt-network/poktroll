#!/usr/bin/env python3
"""
Clean up verbose descriptions in OpenAPI spec by simplifying the standard governance operation descriptions.
"""
import sys
import re
import json

def clean_descriptions_json(content):
    """
    Clean up verbose descriptions in JSON content.
    """
    data = json.loads(content)

    if 'paths' not in data:
        return content

    changes_made = 0

    for path, methods in data['paths'].items():
        for method, operation in methods.items():
            if not isinstance(operation, dict):
                continue

            # Clean up verbose summary
            if 'summary' in operation:
                original = operation['summary']
                # Replace the verbose governance description with a simple one
                cleaned = re.sub(
                    r'UpdateParams defines a \(governance\) operation for updating the module parameters\. The authority defaults to the x/gov module account\.',
                    'Update module parameters via governance.',
                    original
                )
                if cleaned != original:
                    operation['summary'] = cleaned
                    changes_made += 1
                    print(f"Cleaned summary for {path} {method}")

            # Clean up verbose description if it exists
            if 'description' in operation:
                original = operation['description']
                cleaned = re.sub(
                    r'UpdateParams defines a \(governance\) operation for updating the module parameters\. The authority defaults to the x/gov module account\.',
                    'Update module parameters via governance.',
                    original
                )
                if cleaned != original:
                    operation['description'] = cleaned
                    changes_made += 1
                    print(f"Cleaned description for {path} {method}")

    print(f"Cleaned {changes_made} verbose descriptions")
    return json.dumps(data, indent=2)

def clean_descriptions_yaml(content):
    """
    Clean up verbose descriptions in YAML content and set module-specific titles.
    """
    # Find all UpdateParams operations and replace with module-specific titles
    lines = content.split('\n')
    current_path = None
    changes_made = 0

    # Module mapping
    module_mapping = {
        'application': 'application',
        'gateway': 'gateway',
        'migration': 'migration',
        'proof': 'proof',
        'service': 'service',
        'session': 'session',
        'shared': 'shared',
        'supplier': 'supplier',
        'tokenomics': 'tokenomics'
    }

    i = 0
    while i < len(lines):
        line = lines[i]

        # Look for path definitions
        if re.match(r'\s*/pocket\.[^.]+\.(Msg|Query)/.+:$', line):
            current_path = line.strip().rstrip(':')

        # Look for the verbose UpdateParams summary
        elif current_path and 'summary: |-' in line:
            # Check if this is followed by the verbose description
            if i + 2 < len(lines):
                next_line = lines[i + 1].strip() if i + 1 < len(lines) else ""
                line_after = lines[i + 2].strip() if i + 2 < len(lines) else ""

                if ("UpdateParams defines a (governance)" in next_line and
                    "parameters. The authority defaults to the x/gov" in line_after):

                    # Extract module from path
                    module_match = re.match(r'/pocket\.([^.]+)\.(Msg|Query)/(.+)', current_path)
                    if module_match:
                        module_key = module_match.group(1)
                        endpoint = module_match.group(3)
                        module_name = module_mapping.get(module_key, module_key)

                        # Get the operation ID for this path (should be a few lines down)
                        op_id = None
                        for j in range(i + 1, min(i + 10, len(lines))):
                            if 'operationId:' in lines[j]:
                                op_match = re.search(r'operationId:\s*(\S+)', lines[j])
                                if op_match:
                                    op_id = op_match.group(1)
                                    break

                        if op_id:
                            # Replace the multi-line summary with a single line
                            lines[i] = f'      summary: {module_name}.{op_id}'
                            # Remove the next two lines (the verbose description)
                            del lines[i + 1:i + 3]
                            changes_made += 1
                            print(f"Updated summary for {current_path}: {module_name}.{op_id}")

        i += 1

    if changes_made > 0:
        print(f"Updated {changes_made} operation summaries with module-specific titles")
    else:
        print("No verbose descriptions found to clean")

    return '\n'.join(lines)

def main():
    if len(sys.argv) < 2:
        print("Usage: python clean_openapi_descriptions.py <openapi_file>")
        sys.exit(1)

    file_path = sys.argv[1]

    # Read file
    with open(file_path, 'r') as f:
        content = f.read()

    # Detect format and clean
    if file_path.endswith('.json'):
        cleaned_content = clean_descriptions_json(content)
    else:  # YAML
        cleaned_content = clean_descriptions_yaml(content)

    # Write back
    with open(file_path, 'w') as f:
        f.write(cleaned_content)

    print(f"Description cleanup completed for {file_path}")

if __name__ == '__main__':
    main()