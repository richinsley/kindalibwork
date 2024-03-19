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
#
# In /Users/richardinsley/miniconda3/envs/py39/include/python3.9/pyport.h
# we need to append these define overrides at the bottom:
# #define PyAPI_FUNC(RTYPE) RTYPE
# #define PyAPI_DATA(RTYPE) RTYPE
# #define PyMODINIT_FUNC PyObject*
# #define _Py_NO_RETURN
# #define Py_GCC_ATTRIBUTE(x)
# #define Py_DEPRECATED(x)
def preprocess_and_parse(fake_header_file_path: str, header_file_path: str) -> c_ast.FileAST:
    # Path to the fake headers
    fake_libc_include = os.path.join(os.path.dirname(c_parser.__file__), 'fake_libc_include')

    # Run gcc to preprocess the header file, including the fake headers directory
    fake_libc_include = os.path.join(os.path.dirname(c_parser.__file__), 'fake_libc_include')
    cmd = ['gcc', '-E', '-D_POSIX_THREADS', '-DPy_ENABLE_SHARED', '-nostdinc', '-I', fake_header_file_path, header_file_path]
    process = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    stdout, stderr = process.communicate()

    if process.returncode != 0:
        raise RuntimeError(f"gcc preprocessing failed: {stderr.decode()}")

    # Parse the output of gcc
    parser = c_parser.CParser()
    ast = parser.parse(stdout.decode(), filename=header_file_path)

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



# Main function to run the script
def main():
    header_file_path = sys.argv[1] # '/Users/richardinsley/miniconda3/envs/py39/include/python3.9'
    ctags_bin = sys.argv[2] # '/opt/homebrew/bin/ctags'
    fake_header_path = "/Users/richardinsley/Projects/Cradles/gopythoncradle/pycparser/utils/fake_libc_include"

    ctags_output = execute_ctags(ctags_bin, header_file_path)
    header_info = parse_ctags_output(ctags_output)

    # Process header_info as needed
    search_string = "PyAPI_FUNC"
    found_prototypes, depricated_prototypes = find_prototypes_by_search_string(header_info, search_string)
    ast = preprocess_and_parse(fake_header_path, header_file_path + "/Python.h")

    # Create an instance of the visitor with our found prototypes
    finder = FunctionFinder(found_prototypes)
    finder.visit(ast)

    # Convert the functions info to JSON
    json_output = json.dumps(finder.functions_info, indent=4)
    print(json_output)

    # print("Depricated:")
    
    # # Create an instance of the visitor with your found depricated prototypes
    # finder = FunctionFinder(depricated_prototypes)
    # finder.visit(ast)

    # # Print the found functions or process them as needed
    # for func in finder.found_functions:
    #     # You can also convert it back to source code
    #     generator = c_generator.CGenerator()
    #     print(generator.visit(func))

if __name__ == "__main__":
    main()
