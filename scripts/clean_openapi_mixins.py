#!/usr/bin/env python3
"""
Clean up Mixin operation IDs in OpenAPI spec by replacing them with module-specific names.
Works with both JSON and YAML formats without requiring PyYAML.
"""
import sys
import re
import json

def clean_mixin_operations_json(content):
    """
    Replace Mixin operation IDs with module-specific names in JSON content.
    """
    data = json.loads(content)

    if 'paths' not in data:
        return content

    # Module mapping for cleaner names
    module_mapping = {
        'application': 'Application',
        'gateway': 'Gateway',
        'migration': 'Migration',
        'proof': 'Proof',
        'service': 'Service',
        'session': 'Session',
        'shared': 'Shared',
        'supplier': 'Supplier',
        'tokenomics': 'Tokenomics'
    }

    for path, methods in data['paths'].items():
        # Extract module name from path like "/pocket.service.Msg/UpdateParam"
        module_match = re.match(r'/pocket\.([^.]+)\.(Msg|Query)/', path)
        if not module_match:
            continue

        module_key = module_match.group(1)
        module_name = module_mapping.get(module_key, module_key.capitalize())

        for method, operation in methods.items():
            if not isinstance(operation, dict) or 'operationId' not in operation:
                continue

            op_id = operation['operationId']

            # Replace Mixin patterns
            if 'Mixin' in op_id:
                if op_id.startswith('MsgUpdateParam') and not op_id.startswith('MsgUpdateParams'):
                    operation['operationId'] = f'Msg{module_name}UpdateParam'
                elif op_id.startswith('MsgUpdateParams'):
                    operation['operationId'] = f'Msg{module_name}UpdateParams'
                elif op_id.startswith('QueryParams'):
                    operation['operationId'] = f'Query{module_name}Params'
                else:
                    # Generic replacement - remove Mixin and number
                    cleaned = re.sub(r'Mixin\d+', module_name, op_id)
                    operation['operationId'] = cleaned

                print(f"Renamed: {op_id} -> {operation['operationId']}")

    return json.dumps(data, indent=2)

def clean_mixin_operations_yaml(content):
    """
    Replace Mixin operation IDs with module-specific names in YAML content using regex.
    """
    # Extract path -> operationId mappings
    path_operation_map = {}

    # Find all path definitions and their operation IDs
    lines = content.split('\n')
    current_path = None

    for i, line in enumerate(lines):
        # Look for path definitions
        if re.match(r'\s*/pocket\.[^.]+\.(Msg|Query)/.+:$', line):
            current_path = line.strip().rstrip(':')
        # Look for operationId with Mixin
        elif current_path and 'operationId:' in line and 'Mixin' in line:
            op_match = re.search(r'operationId:\s*(\S+)', line)
            if op_match:
                op_id = op_match.group(1)
                path_operation_map[current_path] = op_id

    # Now replace the operation IDs based on their paths
    module_mapping = {
        'application': 'Application',
        'gateway': 'Gateway',
        'migration': 'Migration',
        'proof': 'Proof',
        'service': 'Service',
        'session': 'Session',
        'shared': 'Shared',
        'supplier': 'Supplier',
        'tokenomics': 'Tokenomics'
    }

    for path, old_op_id in path_operation_map.items():
        # Extract module from path
        module_match = re.match(r'/pocket\.([^.]+)\.(Msg|Query)/', path)
        if not module_match:
            continue

        module_key = module_match.group(1)
        module_name = module_mapping.get(module_key, module_key.capitalize())

        # Generate new operation ID
        if old_op_id.startswith('MsgUpdateParam') and not old_op_id.startswith('MsgUpdateParams'):
            new_op_id = f'Msg{module_name}UpdateParam'
        elif old_op_id.startswith('MsgUpdateParams'):
            new_op_id = f'Msg{module_name}UpdateParams'
        elif old_op_id.startswith('QueryParams'):
            new_op_id = f'Query{module_name}Params'
        else:
            # Generic replacement
            new_op_id = re.sub(r'Mixin\d+', module_name, old_op_id)

        # Replace in content
        content = content.replace(f'operationId: {old_op_id}', f'operationId: {new_op_id}')
        print(f"Renamed: {old_op_id} -> {new_op_id}")

    return content

def main():
    if len(sys.argv) < 2:
        print("Usage: python clean_openapi_mixins.py <openapi_file>")
        sys.exit(1)

    file_path = sys.argv[1]

    # Read file
    with open(file_path, 'r') as f:
        content = f.read()

    # Detect format and clean
    if file_path.endswith('.json'):
        cleaned_content = clean_mixin_operations_json(content)
    else:  # YAML
        cleaned_content = clean_mixin_operations_yaml(content)

    # Write back
    with open(file_path, 'w') as f:
        f.write(cleaned_content)

    print(f"Mixin operations cleaned in {file_path}")

if __name__ == '__main__':
    main()