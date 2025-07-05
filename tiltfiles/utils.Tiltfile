# deep_merge_dicts recursively updates the base dictionary with the updates dictionary.
# NOTE: Starlark (tilt) doesn't support multiple kwargs; otherwise, we could simply do `dict(**base, **updates)`.
def deep_merge_dicts(base, updates):
    result = dict(base)  # shallow copy

    for key, updates_value in updates.items():
        base_value = result.get(key)
        # Recurse if both values are dicts
        if type(base_value) == type({}) and type(updates_value) == type({}):
            result[key] = deep_merge_dicts(base_value, updates_value)
        else:
            result[key] = updates_value

    return result

