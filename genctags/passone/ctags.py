import json
import subprocess
import re
import os
import sys
from pycparser import c_parser, c_ast, parse_file, c_generator

# Define a list of structs that we are interested in
# This list is used to filter out the structs that we are interested in
# When adding a new struct to the list, if the struct contains a non-pointer member of a struct type,
# that struct should also be added to the list.  
structlist = ['PyConfig', 'PyPreConfig', 'PyWideStringList', 'PyObject', 'PyMethodDef', 'PyModuleDef_Base', 'PyModuleDef']

# Define the Tag and CHeaderInfo classes
class Tag:
    def __init__(self, tag_type, name, path, pattern, kind, scope, scope_kind):
        self.type = tag_type
        self.name = name
        self.path = path
        self.pattern = pattern
        self.kind = kind
        self.scope = scope
        self.scope_kind = scope_kind

class CHeaderInfo:
    def __init__(self):
        self.macros = []
        self.enumerators = []
        self.enums = []
        self.functions = []
        self.prototypes = []
        self.typedefs = []

    def add_tag(self, tag):
        if tag.kind == "macro":
            self.macros.append(tag)
        elif tag.kind == "enumerator":
            self.enumerators.append(tag)
        elif tag.kind == "enum":
            self.enums.append(tag)
        elif tag.kind == "function":
            self.functions.append(tag)
        elif tag.kind == "prototype":
            self.prototypes.append(tag)
        elif tag.kind == "typedef":
            self.typedefs.append(tag)

# Execute the ctags command
def execute_ctags(ctags_bin, header_file_path):
    cmd = [ctags_bin, "--language-force=C", "--output-format=json", "-R", "--c-kinds=+p", header_file_path]
    try:
        return subprocess.check_output(cmd).decode('utf-8')
    except subprocess.CalledProcessError as e:
        print(f"Error executing ctags: {e}", file=sys.stderr)
        sys.exit(1)

# Parse the output from ctags
def parse_ctags_output(output):
    header_info = CHeaderInfo()
    for line in output.splitlines():
        try:
            tag_data = json.loads(line)
            tag = Tag(tag_data.get('_type'), tag_data.get('name'), tag_data.get('path'),
                      tag_data.get('pattern'), tag_data.get('kind'), tag_data.get('scope'),
                      tag_data.get('scopeKind'))
            header_info.add_tag(tag)
        except json.JSONDecodeError:
            pass  # Ignore lines that are not valid JSON
    return header_info

def find_prototypes_by_search_string(header_info, search_string):
    prototype_names = []
    deprecated_names = []
    # Updated regex with search_string
    regex = rf'{re.escape(search_string)}\([^)]*\)\s*(\w+)\('

    for prototype in header_info.prototypes:
        match = re.search(regex, prototype.pattern)
        if match:
            prototype_name = match.group(1)
            if "Py_DEPRECATED" in prototype.pattern:
                deprecated_names.append(prototype_name)
            else:
                prototype_names.append(prototype_name)

    return prototype_names, deprecated_names

def preprocess_and_parse(fake_header_file_path: str, header_file_path: str) -> c_ast.FileAST:
    pyheader = os.path.join(header_file_path, "Python.h")
    fake_libc_include = os.path.join(os.path.dirname(c_parser.__file__), 'fake_libc_include')

    cmd = None
    if platform == "windows":
        cmd = ['gcc', '-E', '-D_POSIX_THREADS', '-DPy_ENABLE_SHARED', '-DMS_WINDOWS', '-D__int64=int64_t', '-nostdinc', '-I', fake_header_file_path, '-I', fake_libc_include, '-I', header_file_path, pyheader]
    else:
        cmd = ['gcc', '-E', '-D_POSIX_THREADS', '-DPy_ENABLE_SHARED', '-nostdinc', '-I', fake_header_file_path, '-I', fake_libc_include, '-I', header_file_path, pyheader]

    process = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    stdout, stderr = process.communicate()

    if process.returncode != 0:
        raise RuntimeError(f"gcc preprocessing failed: {stderr.decode()}")

    parser = c_parser.CParser()
    ast = parser.parse(stdout.decode(), filename=pyheader)

    return ast

class FunctionFinder(c_ast.NodeVisitor):
    def __init__(self, prototypes):
        self.prototypes = prototypes
        self.functions_info = []

    def visit_Decl(self, node):
        if isinstance(node.type, c_ast.FuncDecl):
            if node.name in self.prototypes:
                func_info = {
                    'name': node.name,
                    'return_type': self.get_type(node.type.type),
                    'parameters': self.get_params(node.type.args)
                }
                self.functions_info.append(func_info)

    def get_type(self, node):
        if isinstance(node, c_ast.TypeDecl):
            return self.get_type(node.type)  # Get the actual type, not the declname
        elif isinstance(node, c_ast.Typename) or isinstance(node, c_ast.Decl):
            return self.get_type(node.type)
        elif isinstance(node, c_ast.PtrDecl):
            return self.get_type(node.type) + '*'
        elif isinstance(node, c_ast.ArrayDecl):
            arr_type = self.get_type(node.type)
            if node.dim:
                return f'{arr_type}[{node.dim.value}]'
            else:
                return f'{arr_type}[]'
        elif isinstance(node, c_ast.IdentifierType):
            return ' '.join(node.names)
        else:
            return 'unknown'

    def get_params(self, param_list):
        if not param_list:
            return []
        params = []
        for param in param_list.params:
            if isinstance(param, c_ast.EllipsisParam):
                params.append({'name': '', 'type': '...'})
            else:
                param_type = self.get_type(param)
                param_name = param.name if param.name else ''
                params.append({'name': param_name, 'type': param_type})
        return params

class StructFinder(c_ast.NodeVisitor):
    def __init__(self):
        self.struct_info = {}
        self.struct_map = {}
        self.function_pointer_map = {}

    def visit_Struct(self, node):
        if node.name:
            self.struct_map[node.name] = node

    def visit_Typedef(self, node):
        if isinstance(node.type, c_ast.TypeDecl) and isinstance(node.type.type, c_ast.IdentifierType):
            struct_name = node.name
            truenode = self.struct_map.get(node.type.type.names[0])
            if struct_name in structlist and truenode is not None:
                members, members_size = self.get_members(truenode.decls)
                self.struct_info[struct_name] = {
                    'name': struct_name,
                    'size': members_size,
                    'members': members,
                    'node': node,
                }
        elif isinstance(node.type, c_ast.TypeDecl) and isinstance(node.type.type, c_ast.Struct):
            struct_node = node.type.type
            struct_name = node.name
            if struct_name in structlist:
                members, members_size = self.get_members(struct_node.decls)
                self.struct_info[struct_name] = {
                    'name': struct_name,
                    'size': members_size,
                    'members': members,
                    'node': node,
                }
        # Check if the typedef is a function pointer
        elif isinstance(node.type, c_ast.PtrDecl) and isinstance(node.type.type, c_ast.FuncDecl):
            typedef_name = node.name
            return_type = self.get_type(node.type.type.type)
            param_types = self.get_param_types(node.type.type.args)
            function_pointer_typedef = {
                'name': typedef_name,
                'return_type': return_type,
                'param_types': param_types
            }
            self.function_pointer_map[typedef_name] = function_pointer_typedef

    def get_param_types(self, param_list):
        if not param_list:
            return []
        param_types = []
        for param in param_list.params:
            if isinstance(param, c_ast.EllipsisParam):
                param_types.append('...')
            else:
                param_type = self.get_type(param.type)
                param_types.append(param_type)
        return param_types
    
    def get_members(self, decls):
        members = []
        offset = 0
        size = 0
        align_to_long = True

        if decls is not None:
            for decl in decls:
                if isinstance(decl, c_ast.Decl):
                    member_type = self.get_type(decl.type)
                    member_name = decl.name
                    if member_name is None:
                        member_name = 'None'
                    member_size = self.get_size(member_type, decl)

                    # if member_type starts with 'union' trun it to 'union'
                    if member_type.startswith('union'):
                        member_type = 'union'

                    member_info = {
                        'name': member_name,
                        'type': member_type,
                        'offset': offset,
                        'size': member_size,
                    }
                    members.append(member_info)
                    offset += member_size
                    size += member_size

                    # if offset is not aligned to long, add padding
                    if member_type == 'PyWideStringList' and align_to_long and offset % 8 != 0:
                        padding = 8 - (offset % 8)
                        offset += padding
                        size += padding

                    if member_size == 4 and  align_to_long and offset % 8 != 0:
                        padding = 8 - (offset % 8)
                        offset += padding
                        size += padding
                elif isinstance(decl, c_ast.Union):
                    # Handle anonymous unions
                    union_members = self.get_union_members(decl)
                    union_size = max(self.get_size(member['type'], decl) for member in union_members)
                    union_info = {
                        'name': '',
                        'type': 'union',
                        'offset': offset,
                        'size': union_size,
                        'members': union_members,
                    }
                    members.append(union_info)
                    offset += union_size
                    size += union_size

        return members, size

    def get_size(self, type_name, decl=None):
        type_sizes = {
            'int': 4,
            'unsigned long': 8,
            'wchar_t *': 8,
            'wchar_t*': 8,
            'wchar_t **': 8,
            'wchar_t**': 8,
            'char *': 8,
            'char*': 8,
            'uint32_t': 4,
            'Py_ssize_t': 8,
            'PyWideStringList': 16,  # Assuming 8 bytes for length and 8 bytes for items pointer
            'PyTypeObject*': 8,  # Assuming 8 bytes for pointer to PyTypeObject. PyTypeObject is an opaque type
        }
        tsize = type_sizes.get(type_name, 0)
        if tsize == 0:
            # a union's size will be the size of the largest member
            if type_name.startswith('union'):
                union_decls_str = type_name.split(None, 1)[1].strip('[]')
                union_members = self.parse_union_members(union_decls_str, decl)
                tsize = 0
                if union_members:
                    for member in union_members[0]:
                        member_size = member['size']
                        if member_size > tsize:
                            tsize = member_size
            # handle types of arrays, ie: uint32_t[4]
            elif '[' in type_name:
                array_type = type_name.split('[')[0]
                array_size = int(type_name.split('[')[1].split(']')[0])
                tsize = type_sizes.get(array_type, 0) * array_size
            # all pointers are 8 bytes
            elif '*' in type_name:
                tsize = 8
            # is it a function pointer typedef
            elif type_name in self.function_pointer_map:
                tsize = 8
            # is it in the struct_info
            elif type_name in self.struct_info:
                if not struct_info_has_zero_member_size(self.struct_info[type_name]):
                    tsize = self.struct_info[type_name]['size']
                else:
                    tsize = 0
            else:
                # an unknown type, return 0
                tsize = 0
        return tsize

    def parse_union_members(self, union_decls_str, decl):
        return self.get_members(decl.type.decls)

    def get_type(self, node):
        if isinstance(node, c_ast.TypeDecl):
            return self.get_type(node.type)  # Get the actual type, not the declname
        elif isinstance(node, c_ast.Typename) or isinstance(node, c_ast.Decl):
            return self.get_type(node.type)
        elif isinstance(node, c_ast.PtrDecl):
            return self.get_type(node.type) + '*'
        elif isinstance(node, c_ast.ArrayDecl):
            arr_type = self.get_type(node.type)
            if node.dim:
                return f'{arr_type}[{node.dim.value}]'
            else:
                return f'{arr_type}[]'
        elif isinstance(node, c_ast.IdentifierType):
            return ' '.join(node.names)
        elif isinstance(node, c_ast.Struct):
            return f'struct {node.name}'
        elif isinstance(node, c_ast.Union):
            if node.name:
                return f'union {node.name}'
            else:
                return f'union {node.decls}'
        elif isinstance(node, c_ast.Enum):
            return f'enum {node.name}'
        else:
            return 'unknown'

def struct_info_has_zero_member_size(struct_info):
    for member in struct_info['members']:
        if member['size'] == 0:
            return True
    return False

# Main function to run the script
def main():
    header_file_path = sys.argv[1]      # Path to the directory containing Python headers
    ctags_bin = sys.argv[2]             # Path to the ctags binary
    fake_header_path = sys.argv[3]      # Path to the fake headers for pycparser

    global platform
    if len(sys.argv) > 4:
        platform = sys.argv[4]          # Platform (e.g., "windows")
    else:
        platform = "linux"

    ctags_output = execute_ctags(ctags_bin, header_file_path)
    header_info = parse_ctags_output(ctags_output)

    # Process header_info as needed
    search_string = "PyAPI_FUNC"
    found_prototypes, deprecated_prototypes = find_prototypes_by_search_string(header_info, search_string)
    ast = preprocess_and_parse(fake_header_path, header_file_path)

    # Create an instance of the visitor with our found prototypes
    finder = FunctionFinder(found_prototypes)
    finder.visit(ast)

    # After parsing the AST with preprocess_and_parse
    struct_finder = StructFinder()
    struct_finder.visit(ast)

    # Create the 'PyStructs' dictionary dynamically
    pystructs_info = {}
    for struct_name in structlist:
        struct_info = struct_finder.struct_info.get(struct_name)
        if struct_info is None:
            continue
        if struct_info['size'] == 0:
            truenode = struct_finder.struct_map.get(struct_info['node'].type.type.name)
            members, members_size = struct_finder.get_members(truenode.decls)
            struct_info = {
                'name': struct_name,
                'size': members_size,
                'members': members,
            }
        struct_finder.struct_info[struct_name] = struct_info
        pystructs_info[struct_name] = struct_info

    # get a list of struct_info that have missing member sizes
    missing_member_size = []
    for struct_name in pystructs_info:
        if struct_info_has_zero_member_size(pystructs_info[struct_name]):
            missing_member_size.append(pystructs_info[struct_name])

    # re-do structs that have missing member sizes
    # full AST parsing may be needed to get the size of the members
    if missing_member_size:
        # walk through the members of the missing structs and find the size of each member
        for struct in missing_member_size:
            offset = 0
            for member in struct['members']:
                member['offset'] = offset
                if member['size'] == 0:
                    # check if the member is a function pointer typedef
                    if member['type'] in struct_finder.function_pointer_map:
                        function_pointer = struct_finder.function_pointer_map[member['type']]
                        member['size'] = 8
                    else:
                        member['size'] = struct_finder.get_size(member['type'])

                offset += member['size']

    missing_member_size = []
    for struct_name in pystructs_info:
        if struct_info_has_zero_member_size(pystructs_info[struct_name]):
            missing_member_size.append(pystructs_info[struct_name])

    # remove the 'node' from all the struct_info
    for struct_name in pystructs_info:
        if 'node' in pystructs_info[struct_name]:
            pystructs_info[struct_name].pop('node')

    combined_info = {
        'PyFunctions': finder.functions_info,
        'PyStructs': pystructs_info
    }

    # Convert the functions and struct info to JSON
    json_output = json.dumps(combined_info, indent=4)
    print(json_output)

if __name__ == "__main__":
    main()
