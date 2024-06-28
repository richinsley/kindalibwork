import json
import subprocess
import re
import os
import sys
from pycparser import c_parser, c_ast
from dagger import DaggerNode, DaggerGraph, DaggerInputPin, DaggerOutputPin

class TypeDaggerGraph(DaggerGraph):
    def __init__(self, id):
        super().__init__(id)

    def get_nodes_with_name(self, name):
        if isinstance(name, dict):
            name = name['name']

        # if the name contains a * then change the name to a pointer
        if '*' in name:
            name = 'pointer'

        # remove trailing whitespace
        name = name.strip()
        nodes = []
        for node in self.nodes:
            if node.name == name:
                nodes.append(node)
        return nodes
    
    def get_node_for_member(self, member):
        if isinstance(member, str):
            tnodes = self.get_nodes_with_name(member)
            if len(tnodes) > 0:
                return tnodes[0]
            else:
                return None
        elif isinstance(member['type'], str):
            name = None
            if member['type'] == 'IdentifierType':
                name = member['name']
            elif member['type'] == 'Enum':
                name = 'Enum'
            else:
                return None
            
            tnodes = self.get_nodes_with_name(name)
            if len(tnodes) > 0:
                return tnodes[0]
            else:
                return None
        elif member['type']['type'] == 'ArrayDecl':
            t = None
            if member['type']['value']['type'] == 'TypeDecl':
                t = member['type']['value']['value']
            else:
                t = member['type']['value']
            tnodes = self.get_nodes_with_name(t['name'])
            if len(tnodes) > 0:
                return tnodes[0]
            else:
                return None
        elif isinstance(member['type']['value'], dict):
            tnodes = self.get_nodes_with_name(member['type']['value']['name'])
            if len(tnodes) > 0:
                return tnodes[0]
            elif member['type']['value']['type'] == 'Enum':
                # handle Enum types
                tnodes = self.get_nodes_with_name('enum')
                if len(tnodes) > 0:
                    return tnodes[0]
                else:
                    return None
            else:
                return None
        else:
            tnodes = self.get_nodes_with_name(member['type']['name'])
            if len(tnodes) > 0:
                return tnodes[0]
            else:
                return None
            
    def gen_sizes(self):
        maxordinals = self.get_max_ordinal(0)
        for i in range(maxordinals):
            nodes = self.get_nodes_with_ordinal(0, i)
            for node in nodes:
                nsize = node.calc_size()
                print(f"Node Size {node.name}: {nsize}")
                
    
## type daggernode
class TypeDaggerNode(DaggerNode):
    # info types:
    # struct
    # typedef
    # intrinsic
    def __init__(self, info):
        super().__init__()
        output_pin = DaggerOutputPin()
        self.set_name(info['name'])
        self.type = info['type']
        self.size = -1
        # if info has 'value' then assign the value to the node
        if 'value' in info:
            self.value = info['value']
        else:
            self.value = None

        # does it have members?
        if 'members' in info and info['members'] is not None and len(info['members']) > 0:
            self.members = info['members']
        else:
            self.members = None
        self.get_output_pins(0).add_pin(output_pin, "output_pin1")

    # check for bitsize on unconnected members
    def _is_bitpacked(self):
        for pin in self.get_input_pins(0).get_all_pins():
            if not pin.is_connected():
                member = pin.member
                if 'bitsize' in member:
                    return True
        return False
    
    def _calc_members(self):
        if self._is_bitpacked():
            bitcount = 0
            for pin in self.get_input_pins(0).get_all_pins():
                bitcount += pin.member['bitsize']
            self.size = (bitcount + 7) // 8
            return self.size
        
        members = {}
        current_offset = 0
        self.size = 0
        for pin in self.get_input_pins(0).get_all_pins():
            # This is where we create a simulated c struct layout
            # We need to calculate the size of the struct and also the offset of each member
            # We will also need to handle padding and alignment.  The size of the member and it's offset
            # will be stored in the members dictionary with the member name as the key.  Track the current
            # offset and add padding as needed to align the next member.  We don't need to worry about
            # unconnected pins because they will be handled by the _is_bitpacked function.
            if pin.is_connected():
                member = pin.member
                tnode = self.parent_graph.get_node_for_member(member)
                if tnode is not None:
                    member_size = tnode.size
                    # if member is an array, calculate the size
                    if isinstance(member['type'], dict) and member['type']['type'] == 'ArrayDecl':
                        member_size = member_size * int(member['type']['dim'])

                    # align the offset
                    align = 8
                    if member_size < align:
                        member_size = align
                    if current_offset % align != 0:
                        current_offset += align - (current_offset % align)

                    newmember = {
                        'name': member['name'],
                        'type': member,
                        'size': member_size,
                        'offset': current_offset
                    }

                    # if member is a pointer, store the pointer type                        
                    if member['type']['name'] == 'pointer':
                        if 'pointer' in member['type']:
                            newmember['pointer'] = member['type']['pointer']

                    members[member['name']] = newmember
                    current_offset += member_size
                    # The size of the struct is the offset of the last member plus the size of the last member
                    self.size = current_offset
                    self.members = members
                else:
                    print(f"Member type not found: {member['type']}")
            else:
                print(f"Member not connected: {pin.pin_name}")
        return self.size
    
    def calc_size(self):
        if self.size != -1:
            return self.size
        
        inputs = self.get_input_pins(0).get_all_pins()
        if self.type == 'Intrinsic':
            # intrinsic types should have a size already
            if self.size <= 0:
                print(f"Intrinsic type has no size: {self.name}")
        elif self.type == 'Struct':
            # calculate the size of the struct
            return self._calc_members()
        elif self.type == 'TypeDecl':
            if self.members != None:
                return self._calc_members()
            elif self.value != None:
                if len(inputs) != 1:
                    print(f"TypeDecl with more than one input: {self.name}")
                else:
                    tnode = self.parent_graph.get_node_for_member(self.value)
                    if tnode is not None:
                        self.size = tnode.size
                    else:
                        print(f"Member type not found: {self.value}")
            else:
                print(f"TypeDecl with no members or value: {self.name}")
        elif self.type == 'Union':
            # calculate the size of the union
            size = 0
            for pin in self.get_input_pins(0).get_all_pins():
                if pin.is_connected():
                    member = pin.member
                    if 'bitsize' in member:
                        size = max(size, member['bitsize'])
                    else:
                        tnode = self.parent_graph.get_node_for_member(member)
                        if tnode is not None:
                            size = max(size, tnode.size)
                        else:
                            print(f"Member type not found: {member['type']}")
            self.size = size
        else:
            print(f"Node type: {self.type}")
        
        return self.size

# Define a list of structs that we are interested in
# This list is used to filter out the structs that we are interested in
# When adding a new struct to the list, if the struct contains a non-pointer member of a struct type,
# that struct should also be added to the list.  
structlist = ['PyConfig', 'PyPreConfig', 'PyMethodDef', 'PyModuleDef', 'PyTypeObject', 'PyObject', 'PyMemberDef', 'PyGetSetDef', 'PyStructSequence_Desc']

# intrinsic types and the sizes for 64-bit systems
intrinsic_types = {
    'char':1,
    'signed char':1,
    'unsigned char':1,
    'short':2,
    'unsigned short':2,
    'signed short':2,
    'int':4,
    'unsigned int':4,
    'signed int':4,
    'long':8,
    'unsigned long':8,
    'signed long':8,
    'long long':8,
    'unsigned long long':8,
    'signed long long':8,
    'float':4,
    'double':8,
    '_Bool':1,
    'Enum':4,
    'pointer':8,
}

# pycparser has incomplete support for typedefs, so we need to manually map some types.  Instances when get_type(node.type) returns "int"
# should use the lookupint function to get the correct type.  This function will return the correct type for the given typedef name.
type_lookup_table = {
"size_t": "unsigned long",
    "__builtin_va_list": "int",
    "__gnuc_va_list": "int",
    "va_list": "int",
    "__int8_t": "signed char",
    "__uint8_t": "unsigned char",
    "__int16_t": "short",
    "__uint16_t": "unsigned short",
    "__int32_t": "int",
    "__uint32_t": "unsigned int",
    "__int64_t": "long",
    "__uint64_t": "unsigned long",
    "__int_least16_t": "short",
    "__uint_least16_t": "unsigned short",
    "__int_least32_t": "int",
    "__uint_least32_t": "unsigned int",
    "__s8": "signed char",
    "__u8": "unsigned char",
    "__s16": "short",
    "__u16": "unsigned short",
    "__s32": "int",
    "__u32": "unsigned int",
    "__s64": "long",
    "__u64": "unsigned long",
    "__dev_t": "int",
    "__uid_t": "int",
    "__gid_t": "int",
    "__off_t": "long",
    "__off64_t": "long",
    "__pid_t": "int",
    "__key_t": "int",
    "__clockid_t": "int",
    "__timer_t": "int",
    "__ssize_t": "int",
    "__mode_t": "unsigned int",
    "__nlink_t": "unsigned int",
    "__fd_mask": "unsigned long",
    "__rlim_t": "unsigned long",
    "__mbstate_t": "int",
    "__flock_t": "int",
    "__iconv_t": "int",
    "__sigset_t": "int",
    "__sigjmp_buf": "int",
    "__jmp_buf": "int",
    "__stack_t": "int",
    "__siginfo_t": "int",
    "__z_stream": "int",
    "int8_t": "signed char",
    "uint8_t": "unsigned char",
    "int16_t": "short",
    "uint16_t": "unsigned short",
    "int32_t": "int",
    "uint32_t": "unsigned int",
    "int64_t": "long",
    "uint64_t": "unsigned long",
    "intptr_t": "long",
    "uintptr_t": "unsigned long",
    "intmax_t": "long",
    "uintmax_t": "unsigned long",
    "int_least8_t": "signed char",
    "uint_least8_t": "unsigned char",
    "int_least16_t": "short",
    "uint_least16_t": "unsigned short",
    "int_least32_t": "int",
    "uint_least32_t": "unsigned int",
    "int_least64_t": "long",
    "uint_least64_t": "unsigned long",
    "int_fast8_t": "signed char",
    "uint_fast8_t": "unsigned char",
    "int_fast16_t": "short",
    "uint_fast16_t": "unsigned short",
    "int_fast32_t": "int",
    "uint_fast32_t": "unsigned int",
    "int_fast64_t": "long",
    "uint_fast64_t": "unsigned long",
    "atomic_int": "int",
    "UsingDeprecatedTrashcanMacro": "int",
    "_LOCK_T": "int",
    "_LOCK_RECURSIVE_T": "int",
    "_off_t": "long",
    "_off64_t": "long",
    "_fpos_t": "long",
    "_ssize_t": "int",
    "wint_t": "unsigned int",
    "_mbstate_t": "int",
    "_flock_t": "int",
    "_iconv_t": "int",
    "__ULong": "unsigned long",
    "__FILE": "int",
    "ptrdiff_t": "long",
    "wchar_t": "int",
    "char16_t": "unsigned short",
    "char32_t": "unsigned int",
    "__loff_t": "long",
    "u_char": "unsigned char",
    "u_short": "unsigned short",
    "u_int": "unsigned int",
    "u_long": "unsigned long",
    "ushort": "unsigned short",
    "uint": "unsigned int",
    "clock_t": "long",
    "time_t": "long",
    "daddr_t": "long",
    "caddr_t": "char *",
    "ino_t": "int",
    "off_t": "long",
    "dev_t": "int",
    "uid_t": "int",
    "gid_t": "int",
    "pid_t": "int",
    "key_t": "int",
    "ssize_t": "int",
    "mode_t": "unsigned int",
    "nlink_t": "unsigned int",
    "fd_mask": "unsigned long",
    "_types_fd_set": "int",
    "clockid_t": "int",
    "timer_t": "int",
    "useconds_t": "unsigned int",
    "suseconds_t": "int",
    "FILE": "int",
    "fpos_t": "long",
    "cookie_read_function_t": "int",
    "cookie_write_function_t": "int",
    "cookie_seek_function_t": "int",
    "cookie_close_function_t": "int",
    "cookie_io_functions_t": "int",
    "div_t": "int",
    "ldiv_t": "long",
    "lldiv_t": "long long",
    "sigset_t": "int",
    "_sig_func_ptr": "int",
    "sig_atomic_t": "int",
    "__tzrule_type": "int",
    "__tzinfo_type": "int",
    "mbstate_t": "int",
    "sem_t": "int",
    "pthread_t": "int",
    "pthread_attr_t": "int",
    "pthread_mutex_t": "int",
    "pthread_mutexattr_t": "int",
    "pthread_cond_t": "int",
    "pthread_condattr_t": "int",
    "pthread_key_t": "int",
    "pthread_once_t": "int",
    "pthread_rwlock_t": "int",
    "pthread_rwlockattr_t": "int",
    "pthread_spinlock_t": "int",
    "pthread_barrier_t": "int",
    "pthread_barrierattr_t": "int",
    "jmp_buf": "int",
    "rlim_t": "int",
    "sa_family_t": "int",
    "sigjmp_buf": "int",
    "stack_t": "int",
    "siginfo_t": "int",
    "z_stream": "int"
}

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
    # write a stub file to include the necessary headers into the header_file_path
    pyheader = os.path.join(header_file_path, "ctags_stub.h")
    stub = """
    #include <Python.h>
    #include <structmember.h>
"""
    with open(pyheader, "w") as f:
        f.write(stub)

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

class PyAPIDataFinder(c_ast.NodeVisitor):
    def __init__(self):
        self.api_data_info = []
        self.api_data_types = ['PyTypeObject', 'PyObject', 'PyMethodDef', 'PyModuleDef', 'PyMemberDef', 'PyGetSetDef', 'PyStructSequence_Desc']
        self.current_depth = 0
        self.files = {}

    def visit_FuncDef(self, node):
        self.current_depth += 1
        self.generic_visit(node)
        self.current_depth -= 1

    def visit_Compound(self, node):
        self.current_depth += 1
        self.generic_visit(node)
        self.current_depth -= 1

    def visit_Decl(self, node):
        # Check if this is a top-level (global) declaration
        if self.current_depth == 0:
            type_name = self.get_type(node.type)
            base_type = type_name.rstrip('* \t')  # Remove trailing '*', spaces, and tabs
            if base_type in self.api_data_types:
                data_info = {
                    'name': node.name,
                    'type': type_name
                }
                # read the line of code from the source file
                if node.coord.file not in self.files:
                    with open(node.coord.file, 'r') as f:
                        self.files[node.coord.file] = f.readlines()

                data_info['line'] = self.files[node.coord.file][node.coord.line - 1]
                # PyAPI_DATA declarations have the form: PyAPI_DATA(PyTypeObject) PyFloat_Type;
                # Use a regex to match the type and name of the declaration from the line and store it in the data_info dictionary
                match = re.match(rf'PyAPI_DATA\({base_type}\)\s+{node.name};', data_info['line'])
                if match:
                    self.api_data_info.append(data_info)
                    print(f"Found potential PyAPI_DATA: {node.name} of type {type_name}")

    def get_type(self, node):
        if isinstance(node, c_ast.TypeDecl):
            return self.get_type(node.type)
        elif isinstance(node, c_ast.IdentifierType):
            return ' '.join(node.names)
        elif isinstance(node, c_ast.PtrDecl):
            return self.get_type(node.type) + '*'
        else:
            return 'unknown'
        
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

class TypeFinder(c_ast.NodeVisitor):
    def __init__(self):
        self.typedef_map = {}
        self.struct_map = {}
        self.discarded_struct_map = {}
        self.function_pointer_map = {}
        self.anon_union_count = 0

    def lookupint(self, node):
        retv = type_lookup_table.get(node.name, "??")
        if retv == "??":
            print(f"Possible incorrect type: {node.name} -> int")
        return retv
    
    def visit_Struct(self, node):
        # and explicit struct declaration without decls is going 
        # to be defined in a Typedef
        if node.name:
            if node.decls:
                members = self.get_members(node.decls)

                # check for nested structs
                for member in members:
                    if member['type']['type'] == 'TypeDecl' and member['type']['value']['type'] == 'Struct' and len(member['type']['value']['members']) > 0:
                        name = member['type']['name']
                        # add to the typedef map
                        self.typedef_map[name] = {
                            'name': name,
                            'type': 'TypeDecl',
                            'value': {
                                'name': name,
                                'type': 'Struct',
                                'members': member['type']['value']['members']
                            }
                        }
                        # convert to an IdentifierType
                        member['type']['value'] = {
                            'name': name,
                            'type': 'IdentifierType',
                            'value': name
                        }
                self.struct_map[node.name] = {
                    'name': node.name,
                    'type': 'Struct',
                    'members': members
                }
            else:
                self.discarded_struct_map[node.name] = {
                    'name': node.name,
                    'type': 'Struct',
                    'members': None
                }
        else:
            print("Anonymous struct")

    def visit_Typedef(self, node):
        if node.name == 'PyMemberDef':
            print("PyMemberDef")

        ntype = self.get_type(node.type)
        # check if we need to lookup the actual int type
        if ntype['type'] == 'TypeDecl' and ntype['value']['name'] == 'int':
            actualtype = self.lookupint(node)
            ntype['value'] = actualtype
            ntype['name'] = actualtype

        # does this typdef already exist?
        if node.name not in self.typedef_map:
            # if it doesn't, add the new name to the list
            self.typedef_map[node.name] = ntype
        else:
            print(f"Typedef already exists: {node.name}")

            
    # get type information from the node
    def get_type(self, node):
        if isinstance(node, c_ast.TypeDecl):
            t = self.get_type(node.type)

            # if this is a struct, check members for a nested struct definitions or unions
            # these can be identified by having a type of 'TypeDecl' and a value type of 'Struct'
            if t['type'] == 'Struct' and t['members'] is not None and len(t['members']) > 0:
                for member in t['members']:
                    if member['type']['type'] == 'TypeDecl' and member['type']['value']['type'] == 'Struct' and len(member['type']['value']['members']) > 0:
                        name = member['type']['name']
                        # add to the typedef map
                        self.typedef_map[name] = {
                            'name': name,
                            'type': 'TypeDecl',
                            'value': {
                                'name': name,
                                'type': 'Struct',
                                'members': member['type']['value']['members']
                            }
                        }

                        # convert to an IdentifierType
                        return {
                            'name': name,
                            'type': 'IdentifierType',
                            'value': name
                        }
                    elif member['type']['type'] == 'TypeDecl' and member['type']['value']['type'] == 'Union' and len(member['type']['value']['members']) > 0:
                        name = member['type']['name']
                        
                        # Handle anonymous unions
                        if name is None:
                            name = f"{member['name']}_anon_union_{self.anon_union_count}"
                            self.anon_union_count += 1

                        # add to the typedef map
                        self.typedef_map[name] = {
                            'name': name,
                            'type': 'TypeDecl',
                            'value': {
                                'name': name,
                                'type': 'Union',
                                'members': member['type']['value']['members']
                            }
                        }
                        # convert to an IdentifierType
                        return {
                            'name': name,
                            'type': 'IdentifierType',
                            'value': name
                        }
                    
            return {
                'name': node.declname,
                'type': "TypeDecl",
                'value': t
            }
        # elif isinstance(node, c_ast.Typename) or isinstance(node, c_ast.Decl):
        #     # not hit
        #     return self.get_type(node.type)
        elif isinstance(node, c_ast.PtrDecl):
            t = self.get_type(node.type)
            retv = {
                'name': "pointer",
                'type': "IdentifierType",
                'value': "pointer"
            }
            if 'pointer' in t:
                # we already have a pointer type, so just return it
                retv['pointer'] = t['pointer']
            elif 'type' in t:
                # store the pointer type if not already stored
                if t['type'] == 'FuncDecl':
                    # store the function pointer type
                    retv['pointer'] = 'function'
                elif isinstance(t['value'], dict):
                    retv['pointer'] = t['value']['name']
                else:
                    retv['pointer'] = t['name']
            
            return retv
        elif isinstance(node, c_ast.ArrayDecl):
            dim = 0
            arr_type = self.get_type(node.type)
            if node.dim:
                dim = node.dim.value

            return {
                'name': arr_type['name'],
                'type': "ArrayDecl",
                'value': arr_type,
                'dim': dim,
            }
        elif isinstance(node, c_ast.IdentifierType):
            name = ' '.join(node.names)
            return {
                'name': name,
                'type': 'IdentifierType',
                'value': name
            }
            return ' '.join(node.names)
        elif isinstance(node, c_ast.Struct):
            members = []
            if node.decls is not None:
                members = self.get_members(node.decls)
                # TODO add struct to struct_map if members are not empty
            return {
                'name': node.name,
                'type': 'Struct',
                'members': members
            }

        elif isinstance(node, c_ast.Union):
            members = []
            if node.decls is not None:
                members = self.get_members(node.decls)
                
                # If any members are a struct typedef with members, add it to the struct map
                for member in members:
                    if member['type']['type'] == 'TypeDecl' and member['type']['value']['type'] == 'Struct':
                        name = member['type']['name']
                        # add to the typedef map
                        self.typedef_map[name] = {
                            'name': name,
                            'type': 'TypeDecl',
                            'value': {
                                'name': name,
                                'type': 'Struct',
                                'members': member['type']['value']['members']
                            }
                        }
                        # convert member to an IdentifierType to point to the typedef and add back to members
                        members[members.index(member)]['type']['value'] = {
                            'name': name,
                            'type': 'IdentifierType',
                            'value': name
                        }
                    
                return {
                    'name': node.name,
                    'type': 'Union',
                    'members': members
                }
        elif isinstance(node, c_ast.Enum):
            return {
                'name': 'Enum',
                'type': 'Enum',
            }
        # a function pointer typedef?
        elif isinstance(node, c_ast.FuncDecl):
            # get the function name
            name = None
            # does node.type have a declname member?
            if hasattr(node.type, 'declname'):
                name = node.type.declname
            else:
                name = node.type.type.declname
                
            return {
                'name': name,
                'type': 'FuncDecl',
                ## todo add return type and parameters
            }
        else:
            return {
                'name': node.name,
                'type': 'Unknown',
            }
    

    # typedecl
    # identifier
    # ptrdecl
    # struct
    # enum
    # funcdecl
    # union
    # arraydecl

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
                    if member_type['type'] == 'Union':
                        # Handle anonymous unions
                        if member_name is None:
                            member_name = f"anon_union_{self.anon_union_count}"
                            self.anon_union_count += 1

                        union_members = self.get_members(decl.type.decls)
                        # add to the typedef map
                        self.typedef_map[member_name] = {
                            'name': member_name,
                            'type': 'TypeDecl',
                            'value': {
                                'name': member_name,
                                'type': 'Union',
                                'members': union_members
                            }
                        }

                        # convert to an IdentifierType member
                        member_info = {
                            'name': member_name,
                            'type': {
                                'name': member_name,
                                'type': 'IdentifierType',
                                'value': member_name
                            },
                            'offset': -1,
                            'size': -1,
                        }
                    else:
                        if member_name is None:
                            member_name = 'None'

                        member_info = {
                            'name': member_name,
                            'type': member_type,
                            'offset': -1,
                            'size': -1,
                        }

                        # check if a bitsize is defined
                        if decl.bitsize is not None:
                            member_info['bitsize'] = int(decl.bitsize.value)

                    members.append(member_info)

                elif isinstance(decl, c_ast.Union):
                    # Handle anonymous unions
                    union_members = self.get_members(decl)
                    union_size = max(self.get_size(member['type'], decl) for member in union_members)
                    union_info = {
                        'name': '',
                        'type': 'union',
                        'offset': -1,
                        'size': -1,
                        'members': union_members,
                    }
                    members.append(union_info)

        return members
    
    def find_typedef_for_struct(self, name):
        if isinstance(name, dict):
            name = name['name']

        for key, value in self.typedef_map.items():
            if isinstance(value, dict) and value['type'] == 'TypeDecl': 
                if isinstance(value['value'], dict) and  value['value']['type'] == 'Struct' and value['value']['name'] == name:
                    return value
        return None
    
def proc_node_members(graph, node):
    if node.members != None:
        # add an input pin for each member
        for member in node.members:
            input_pin = DaggerInputPin()
            input_pin.set_pin_name(member['name'])
            input_pin.member = member
            node.get_input_pins(0).add_pin(input_pin, member['name'])
            # if member has a bitsize, set the pin bitsize property and don't connect
            if 'bitsize' in member:
                input_pin.bitsize = member['bitsize']
            else:
            # find the node for the member type
                tnode = graph.get_node_for_member(member)
                if tnode is not None:
                    # our output pin collection always has one pin
                    opin = tnode.get_output_pins(0).get_all_pins()[0]
                    if not opin.connect_to_input(input_pin):
                        print(f"Failed to connect input pin: {input_pin.name}")
                else:
                    print(f"Member type not found: {member['type']}")

# Main function to run the script
def main():
    header_file_path = sys.argv[1]      # Path to the directory containing Python headers
    ctags_bin = sys.argv[2]             # Path to the ctags binary
    fake_header_path = sys.argv[3]      # Path to the fake headers for pycparser
    global platform
    platform = sys.argv[4]              # Platform (e.g., "windows", "darwin", "linux)
    output_path = sys.argv[5]           # Path to the output file

    ctags_output = execute_ctags(ctags_bin, header_file_path)
    header_info = parse_ctags_output(ctags_output)

    # Process header_info as needed
    search_string = "PyAPI_FUNC"
    found_prototypes, deprecated_prototypes = find_prototypes_by_search_string(header_info, search_string)
    ast = preprocess_and_parse(fake_header_path, header_file_path)

    # Create an instance of the visitor with our found prototypes
    finder = FunctionFinder(found_prototypes)
    finder.visit(ast)

    # Create an instance of PyAPIDataFinder
    api_data_finder = PyAPIDataFinder()
    api_data_finder.visit(ast)

    # After parsing the AST with preprocess_and_parse
    type_finder = TypeFinder()
    type_finder.visit(ast)

    # find all typedefs that are structs with members
    struct_typedefs = {}
    struct_empty_typedefs = {}
    for key, value in type_finder.typedef_map.items():
        if value['type'] == 'TypeDecl' and isinstance(value['value'], dict):
            t = value['value']
            if t['type'] == 'Struct' and t['members'] is not None:
                if len(t['members']) > 0:
                    # a struct typedef with members
                    t['origin'] = None
                elif t['name'] in type_finder.struct_map:
                    # a struct typedef with no members
                    # and the struct is in the struct map
                    t['members'] = type_finder.struct_map[t['name']]['members']
                    # remove the struct from the struct map
                    del type_finder.struct_map[t['name']]
                elif key in type_finder.struct_map:
                    # unlikely to hit this
                    struct_typedefs[key] = t
                    # remove the struct from the struct map
                    del type_finder.struct_map[key]
                else:
                    # an orphaned struct typedef
                    # this should be ok if references to it in typedefs are pointers
                    struct_empty_typedefs[key] = t
                    print(f"Struct typedef {key} has no members and is not in struct map")

    # process the graph
    print("Processing the graph...")

    # create the graph
    graph = TypeDaggerGraph(1)

    # disable topoplogy
    graph.set_enable_topology(False)

    # create nodes for the intrinsic types
    for key, value in intrinsic_types.items():
        intrinsic_info = {
            'name': key,
            'type': 'Intrinsic',
            'size': value,
            'offset': -1
        }
        node = TypeDaggerNode(intrinsic_info)
        node.size = value
        graph.add_node(node)

    # create nodes for the remaining struct typedefs in the struct map
    for key, value in type_finder.struct_map.items():
        struct_info = {
            'name': key,
            'type': 'Struct',
            'members': value['members']
        }
        node = TypeDaggerNode(struct_info)
        graph.add_node(node)

    # create nodes for the struct typedefs
    for key, value in type_finder.typedef_map.items():
        typedef_info = {
            'name': key,
            'type': 'TypeDecl',
            'value': value['value'],
            'members': None
        }
        if isinstance(value['value'], dict) and 'members' in value['value']:
            typedef_info['members'] = value['value']['members']
            # change TypeDecl to Union if the typedef is a union
            if value['value']['type'] == 'Union':
                typedef_info['type'] = 'Union'
        node = TypeDaggerNode(typedef_info)
        graph.add_node(node)

    # process the nodes
    for node in graph.nodes:
        if node.name == 'atomic_bool':
            print(f"found atomic_bool")
        if node.name == 'char':
            print(f"found char *")

        print(f"Processing node: {node.name}")
        if node.type == "Intrinsic":
            node.size = intrinsic_types[node.name]
        elif node.type == "Struct":
            # add an input pin for each member
            proc_node_members(graph, node)
        elif node.type == "TypeDecl":
            # typedecl will have members or value
            if node.members != None:
                # add an input pin for each member
                proc_node_members(graph, node)
            elif node.value != None:
                tnode = None
                tnodes = graph.get_nodes_with_name(node.value)
                if len(tnodes) == 0:
                    # check for edge case where the target member is a struct that was typedef'd
                    foundtype = type_finder.find_typedef_for_struct(node.value)
                    if foundtype is not None:
                        tnodes = graph.get_nodes_with_name(foundtype)
                        if len(tnodes) > 0:
                            tnode = tnodes[0]
                else:
                    tnode = tnodes[0]

                if tnode is not None:
                    opin = tnode.get_output_pins(0).get_all_pins()[0]
                    # add an input pin for the typedecl
                    input_pin = DaggerInputPin()
                    input_pin.set_pin_name(node.name)
                    input_pin.member = None
                    node.get_input_pins(0).add_pin(input_pin, node.name)
                    if not opin.connect_to_input(input_pin):
                        print(f"Failed to connect input pin: {input_pin.name}")
                else:
                    print(f"Member node type not found: {node.value}")
            else:
                print(f"Typedecl with no members or value: {node.name}")
        elif node.type == "Union":
            # add an input pin for each member
            proc_node_members(graph, node)
        else:
            print(f"Node type: {node.type}")

    # re-enable topoplogy
    graph.set_enable_topology(True)

## When calculating sizes:
##      Check unconnected pins for bitsize!!! (PyASCIIObject)

    subgraph_count = graph.get_sub_graph_count(0)
    print("Graph subgraph count: ", subgraph_count)

    print("Orphan Nodes:")
    orphan_nodes = []
    for i in range(subgraph_count):
        nodes = graph.get_sub_graph_nodes(0, i)
        if len(nodes) == 1:
            n = nodes[0]
            print(f"... Node: {n.name}")
            orphan_nodes.append(n)

    # remove orphan nodes
    graph.set_enable_topology(False)
    for n in orphan_nodes:
        graph.remove_node(n)
    graph.set_enable_topology(True)

    # # print the graph
    # print("Graph:")
    # for node in graph.get_nodes():
    #     print(f"Node: {node.get_instance_id()}, Name: {node.get_name()}")
    #     for pin in node.get_input_pins(0).get_all_pins():
    #         print(f"  Input Pin: {pin.get_pin_name()}, Connected: {pin.is_connected()}")
    #     for pin in node.get_output_pins(0).get_all_pins():
    #         print(f"  Output Pin: {pin.get_pin_name()}, Connected to: {[p.get_instance_id() for p in pin.get_connected_to()]}")

    # generate sizes
    graph.gen_sizes()

    # build the PyStructs structlist for the required structs
    PyStructs = {}
    for struct in structlist:
        if struct not in PyStructs:
            struct_node = graph.get_node_for_member(struct)
            struct_ascendents = struct_node.get_ascendents(0)
            # append struct_node to struct_ascendents
            struct_ascendents.append(struct_node)
            for node in struct_ascendents:
                if node.name not in PyStructs:
                    if node.members != None and len(node.members) > 0:
                        members = []
                        items = []
                        if hasattr(node.members, 'items'):
                            ditems = node.members.items()
                            for key, member in ditems:
                                items.append(member)
                        else:
                            for member in node.members:
                                items.append(member)

                        for member in items:
                            newmember = {
                                'name': member['name'],
                                'offset': member['offset'],
                                'size': member['size'],
                            }

                            if 'pointer' in member:
                                newmember['type'] = 'pointer'
                                newmember['pointer_type'] = member['pointer']
                            elif 'pointer' in member['type']:
                                newmember['type'] = 'pointer'
                                newmember['pointer_type'] = member['type']['pointer']
                            elif 'pointer' in member['type']['type']:
                                newmember['type'] = 'pointer'
                                newmember['pointer_type'] = member['type']['type']['pointer']
                            else:
                                newmember['type'] = member['type']['name']

                            if 'bitsize' in member:
                                newmember['bitsize'] = member['bitsize']
                            
                            members.append(newmember)
                        sinfo = {
                            'name': node.name,
                            'size': node.size,
                            'members': members
                        }
                        PyStructs[node.name] = sinfo

    PyData = {}
    for data in api_data_finder.api_data_info:
        PyData[data['name']] = data['type']

    combined_info = {
        'PyFunctions': finder.functions_info,
        'PyStructs': PyStructs,
        'PyData': PyData,
    }

    # Convert the functions and struct info to JSON
    json_output = json.dumps(combined_info, indent=4)
    
    # write the JSON to /Users/richardinsley/Projects/comfycli/kindalib/pkg/platform_ctags/darwin/ctags-311.json
    with open(output_path, 'w') as f:
        f.write(json_output)

    print("YAY! Done!")
if __name__ == "__main__":
    main()
