import json
import subprocess
import re
import os
import sys
from pycparser import c_parser, c_ast, parse_file, c_generator

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
    return subprocess.check_output(cmd).decode('utf-8')

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
    depricated_names = []
    # Updated regex with search_string
    regex = rf'{re.escape(search_string)}\([^)]*\)\s*(\w+)\('

    for prototype in header_info.prototypes:
        match = re.search(regex, prototype.pattern)
        if match:
            # currently include functions with Py_DEPRECATED
            prototype_name = match.group(1)
            if "Py_DEPRECATED" in prototype.pattern:
                depricated_names.append(prototype_name)
            else:
                prototype_names.append(prototype_name)

    return prototype_names, depricated_names

# we need to clone the pycparser repo and point to the fake headers directory
# git clone https://github.com/eliben/pycparser.git

def preprocess_and_parse(fake_header_file_path: str, header_file_path: str) -> c_ast.FileAST:
    pyheader = os.path.join(header_file_path, "Python.h")
    # Path to the fake headers
    fake_libc_include = os.path.join(os.path.dirname(c_parser.__file__), 'fake_libc_include')

    # Run gcc to preprocess the header file, including the fake headers directory
    fake_libc_include = os.path.join(os.path.dirname(c_parser.__file__), 'fake_libc_include')
    cmd = None
    # fake_header_file_path: /Users/richardinsley/Projects/comfycli/kindalib/genctags/pycparser/utils/fake_libc_include
    # fake_libc_include: /Users/richardinsley/Projects/comfycli/kindalib/genctags/micromamba/envs/myenv310/lib/python3.10/site-packages/pycparser/fake_libc_include
    # header_file_path: /Users/richardinsley/miniconda3/envs/py310/include/python3.10
    # pyheader: /Users/richardinsley/miniconda3/envs/py310/include/python3.10/Python.h

    # fake_header_file_path: /Users/richardinsley/Projects/comfycli/kindalib/genctags/pycparser/utils/fake_libc_include
    # fake_libc_include: /Users/richardinsley/Projects/comfycli/kindalib/genctags/micromamba/envs/myenv310/lib/python3.10/site-packages/pycparser/fake_libc_include
    # header_file_path: /Users/richardinsley/Projects/comfycli/kindalib/genctags/micromamba/envs/myenv310/include/python3.10
    # pyheader: /Users/richardinsley/Projects/comfycli/kindalib/genctags/micromamba/envs/myenv310/include/python3.10/Python.h
    
    # print(f"fake_header_file_path: {fake_header_file_path}")
    # print(f"fake_libc_include: {fake_libc_include}")
    # print(f"header_file_path: {header_file_path}")
    # print(f"pyheader: {pyheader}")
    if platform == "windows":
        cmd = ['gcc', '-E', '-D_POSIX_THREADS', '-DPy_ENABLE_SHARED', '-DMS_WINDOWS', '-D__int64=int64_t', '-nostdinc', '-I', fake_header_file_path, '-I', fake_libc_include, '-I', header_file_path, pyheader]
    else:
        cmd = ['gcc', '-E', '-D_POSIX_THREADS', '-DPy_ENABLE_SHARED', '-nostdinc', '-I', fake_header_file_path, '-I', fake_libc_include, '-I', header_file_path, pyheader]

    process = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    stdout, stderr = process.communicate()

    if process.returncode != 0:
        raise RuntimeError(f"gcc preprocessing failed: {stderr.decode()}")

    # Parse the output of gcc
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
        # Add more cases as needed for other types
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

    def visit_Typedef(self, node):
        if isinstance(node.type, c_ast.TypeDecl) and isinstance(node.type.type, c_ast.Struct):
            struct_node = node.type.type
            struct_name = node.name
            if node.name == 'PyConfig' or node.name == 'PyPreConfig' or node.name == 'PyWideStringList':
                members, members_size = self.get_members(struct_node.decls)
                self.struct_info[struct_name] = {
                    'name': node.name,
                    'size': members_size,
                    'members': members,
                }

    def get_members(self, decls):
        members = []
        offset = 0
        size = 0
        align_to_long = True

        for decl in decls:
            if isinstance(decl, c_ast.Decl):
                member_type = self.get_type(decl.type)
                member_name = decl.name
                member_size = self.get_size(member_type)
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


        return members, size

    def get_size(self, type_name):
        # Define the size of each type in bytes
        # There is no struct packing specified in the Pyton codebase, so we assume int is 4 bytes, and non-aligned
        type_sizes = {
            'int': 4,
            'unsigned long': 8,
            'wchar_t *': 8,
            'wchar_t*': 8,
            'wchar_t **': 8,
            'wchar_t**': 8,
            'char *': 8,
            'char*': 8,
            # Py_ssize_t is a signed integral type such that sizeof(Py_ssize_t) ==
            # sizeof(size_t).  C99 doesn't define such a thing directly (size_t is an
            # unsigned integral type).  See PEP 353 for details.
            'Py_ssize_t': 8,
            'PyWideStringList': 16,  # Assuming 8 bytes for length and 8 bytes for items pointer
            # Add more type sizes as needed
        }
        tsize = type_sizes.get(type_name, 0)
        if tsize == 0:
            print(f"Unknown type: {type_name}")
        return tsize

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
            return f'union {node.name}'
        elif isinstance(node, c_ast.Enum):
            return f'enum {node.name}'
        # Add more cases as needed for other types
        else:
            return 'unknown'
    
# Main function to run the script
def main():
    header_file_path = sys.argv[1]      # '/Users/richardinsley/miniconda3/envs/py39/include/python3.10'
    ctags_bin = sys.argv[2]             # '/opt/homebrew/bin/ctags'
    fake_header_path = sys.argv[3]      # "/Users/richardinsley/Projects/Cradles/gopythoncradle/pycparser/utils/fake_libc_include"

    global platform
    if len(sys.argv) > 4:
        platform = sys.argv[4]              # "windows"
    else:
        platform = "linux"

    ctags_output = execute_ctags(ctags_bin, header_file_path)
    header_info = parse_ctags_output(ctags_output)

    # Process header_info as needed
    search_string = "PyAPI_FUNC"
    found_prototypes, depricated_prototypes = find_prototypes_by_search_string(header_info, search_string)
    ast = preprocess_and_parse(fake_header_path, header_file_path)

    # Create an instance of the visitor with our found prototypes
    finder = FunctionFinder(found_prototypes)
    finder.visit(ast)

    # After parsing the AST with preprocess_and_parse
    struct_finder = StructFinder()
    struct_finder.visit(ast)

    # Access the struct info, e.g., for PyConfig
    pyconfig_info = struct_finder.struct_info.get('PyConfig')
    pypreconfig_info = struct_finder.struct_info.get('PyPreConfig')
    pywidestringlist_info = struct_finder.struct_info.get('PyWideStringList')
    # if pyconfig_info:
    #     print(json.dumps(pyconfig_info, indent=4))
    # if pypreconfig_info:
    #     print(json.dumps(pypreconfig_info, indent=4))
    # if pywidestringlist_info:
    #     print(json.dumps(pywidestringlist_info, indent=4))

    combined_info = {
        'PyFunctions': finder.functions_info,
        'PyStructs': {
            'PyConfig': pyconfig_info,
            'PyPreConfig': pypreconfig_info,
            'PyWideStringList': pywidestringlist_info
        }
    }
    
    # Convert the functions info to JSON
    json_output = json.dumps(combined_info, indent=4)
    print(json_output)

if __name__ == "__main__":
    main()
