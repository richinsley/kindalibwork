import uuid
from functools import cmp_to_key

class DaggerBase:
    def __init__(self):
        self.instance_id = str(uuid.uuid4())
        self.parent = None

    def get_instance_id(self):
        return self.instance_id

    def get_parent(self):
        return self.parent

    def set_parent(self, parent):
        self.parent = parent

    def emit_error(self, err):
        print(f"Error: {err}")

    def purge_all(self):
        pass

class DaggerSignal:
    def __init__(self):
        self.callbacks = []

    def connect(self, slot):
        self.callbacks.append(slot)

    def disconnect(self, slot):
        self.callbacks = [cb for cb in self.callbacks if cb != slot]

    def disconnect_all(self):
        self.callbacks = []

    def emit(self, *args):
        for cb in self.callbacks:
            cb(*args)

class PinDirection:
    Unknown = "Unknown"
    Input = "Input"
    Output = "Output"

class DaggerBasePin(DaggerBase):
    def __init__(self, direction):
        super().__init__()
        self.pin_name = ""
        self.parent_node = None
        self.direction = direction
        self.name_set = False
        self._can_rename = False
        self.max_auto_clone = 0
        self.auto_clone_count = 0
        self.auto_clone_ref = 0
        self.auto_clone_master = None
        self.auto_clone_name_template = ""
        self.original_name = ""
        self.parent_node_changed = DaggerSignal()
        self.parent_graph_changed = DaggerSignal()
        self.pin_name_changed = DaggerSignal()
        self.can_rename_changed = DaggerSignal()
        self.pin_connected = DaggerSignal()
        self.pin_disconnected = DaggerSignal()

    def get_direction(self):
        return self.direction

    def get_pin_name(self):
        return self.pin_name

    def set_pin_name(self, name):
        self.pin_name = name
        if not self.name_set:
            self.name_set = True
            self.original_name = name

        if self.parent_node:
            self.pin_name_changed.emit()

    def get_parent_node(self):
        return self.parent_node

    def set_parent_node(self, node):
        self.parent_node = node
        self.parent_node_changed.emit()

    def is_input_pin(self):
        return self.direction == PinDirection.Input

    def get_auto_clone_ref_count(self):
        return self.auto_clone_ref

    def set_auto_clone_master(self, master):
        self.auto_clone_master = master

    def get_auto_clone_count(self):
        return self.auto_clone_count

    def get_max_auto_clone(self):
        return self.max_auto_clone

    def get_auto_clone_name_template(self):
        return self.auto_clone_name_template

    def get_index(self):
        parent_collection = self.get_parent()
        return parent_collection.get_index(self)

    def get_auto_clone_master(self):
        return self.auto_clone_master

    def is_auto_cloned(self):
        return self.auto_clone_master is not None and self.auto_clone_master != self

    def can_rename(self):
        return self._can_rename

    def set_can_rename(self, val):
        self._can_rename = val
        if self.parent_node:
            self.can_rename_changed.emit()

    def get_topology_system(self):
        return self.get_parent().get_topology_system()

    def is_connected(self):
        return False

    def get_original_name(self):
        return self.original_name

    def can_connect_to_pin(self, pin):
        return self.get_direction() != pin.get_direction()

    def get_auto_clone(self):
        return self.auto_clone_master == self

    def set_auto_clone(self, max_auto_clone_count, auto_clone_name_template):
        self.max_auto_clone = max_auto_clone_count
        self.auto_clone_name_template = auto_clone_name_template
        self.auto_clone_master = self
        return True

    def inc_auto_clone_count(self):
        self.auto_clone_count += 1
        self.auto_clone_ref += 1

    def dec_auto_clone_count(self):
        self.auto_clone_count -= 1

    def gen_cloned_name_from_template(self):
        rcount = str(self.auto_clone_master.get_auto_clone_ref_count())
        new_name = self.auto_clone_master.get_auto_clone_name_template().replace("%", rcount)
        self.set_pin_name(new_name)

    def cloned(self, from_master):
        self.auto_clone_master = from_master
        self.auto_clone_master.inc_auto_clone_count()
        self.set_can_rename(from_master.can_rename())
        self.gen_cloned_name_from_template()

    def clone(self):
        if self.auto_clone_master is None:
            return None

        if self.auto_clone_master.get_direction() == PinDirection.Input:
            npin = DaggerInputPin()
            return npin

        return None

    def on_removed(self):
        pass

    def on_cloned(self):
        pass

    def purge_all(self):
        super().purge_all()

class DaggerInputPin(DaggerBasePin):
    def __init__(self):
        super().__init__(PinDirection.Input)
        self.connected_to = None

    def get_direction(self):
        return PinDirection.Input

    def get_connected_to(self):
        return self.connected_to

    def set_connected_to(self, pin):
        self.connected_to = pin

    def get_connected_to_uuid(self):
        if self.is_connected():
            return self.connected_to.get_instance_id()
        return "00000000-0000-0000-0000-000000000000"

    def is_connected(self):
        return self.connected_to is not None

    def can_connect_to_pin(self, pin):
        tsystem = self.get_topology_system()
        if tsystem != pin.get_topology_system():
            self.emit_error("pins must belong to the same topology system")
            return False

        if self.parent_node is None:
            return super().can_connect_to_pin(pin)

        if self.parent_node == pin.get_parent_node():
            self.emit_error("pins belong to the same parent node")
            return False

        retv = False
        if self.parent_node.get_parent_graph().get_enable_topology():
            if not self.parent_node.is_descendent_of(pin.get_parent_node(), tsystem):
                retv = super().can_connect_to_pin(pin)
            elif pin.get_parent_node().get_ordinal(tsystem) <= self.parent_node.get_ordinal(tsystem):
                retv = super().can_connect_to_pin(pin)
        else:
            retv = True
        return retv

    def disconnect_pin(self, force_disconnect):
        if self.parent_node is None:
            return False

        if self.is_connected():
            return self.connected_to.disconnect_pin(self, force_disconnect)

        return True

    def set_max_auto_clone(self, max_auto_clone_count):
        self.max_auto_clone = max_auto_clone_count

    def purge_all(self):
        super().purge_all()
        self.connected_to = None

class DaggerOutputPin(DaggerBasePin):
    def __init__(self):
        super().__init__(PinDirection.Output)
        self.connected_to = []
        self.allow_multi_connect = True

    def get_direction(self):
        return PinDirection.Output

    def get_connected_to(self):
        return self.connected_to

    def allow_multi_connect(self):
        return self.allow_multi_connect

    def set_allow_multi_connect(self, val):
        self.allow_multi_connect = val

    def is_connected(self):
        return len(self.connected_to) != 0

    def get_connected_to_uuids(self):
        return [pin.get_instance_id() for pin in self.connected_to]

    def connect_to_input(self, input_pin):
        if input_pin is None:
            self.emit_error("Input pin was null in ConnectToInput")
            return False

        output_pin_node = self.get_parent_node()
        if output_pin_node is None:
            self.emit_error("Output pin is not associated with a DaggerNode")
            return False
        output_pin_container = output_pin_node.get_parent_graph()

        input_pin_node = input_pin.get_parent_node()
        if input_pin_node is None:
            self.emit_error("Input pin is not associated with a DaggerNode")
            return False
        input_pin_container = input_pin.get_parent_node().get_parent_graph()

        if output_pin_container is None:
            self.emit_error("Output pin is not associated with a DaggerNode or DaggerGraph")
            return False

        if input_pin_container is None:
            self.emit_error("Input pin is not associated with a DaggerNode or DaggerGraph")
            return False

        if input_pin_container != output_pin_container:
            self.emit_error("Input pin and Output pin are not associated with the same DaggerGraph")
            return False

        if output_pin_container.get_enable_topology():
            if not input_pin.can_connect_to_pin(self):
                self.emit_error("Input pin indicates it cannot connect to this output pin")
                return False

            if not self.can_connect_to_pin(input_pin):
                self.emit_error("Parent node indicates Input pin cannot connect to this output pin")
                return False

        if input_pin.is_connected():
            if input_pin.get_auto_clone_master() is not None:
                self.emit_error("cannot swap connections on cloned pins")
                return False

            if not input_pin.disconnect_pin(False):
                self.emit_error("Input pin is already connected and was not allowed to disconnect")
                return False

        if output_pin_container.before_pins_connected(self, input_pin):
            self.connected_to.append(input_pin)
            input_pin.set_connected_to(self)

            output_pin_container.on_pins_connected(self, input_pin)

            output_pin_container.after_pins_connected(self, input_pin)

            return True

        return False

    def can_connect_to_pin(self, pin):
        if pin is None or pin.get_direction() != PinDirection.Input:
            return False

        mtop = self.get_topology_system()
        ttop = pin.get_topology_system()
        if mtop != ttop:
            return False

        if pin.is_connected():
            return False

        if not self.allow_multi_connect and self.is_connected():
            return False

        if not pin.get_parent_node().is_descendent_of(self.parent_node, mtop):
            return super().can_connect_to_pin(pin)
        elif pin.get_parent_node().get_ordinal(mtop) >= self.parent_node.get_ordinal(mtop):
            return super().can_connect_to_pin(pin)

        return False

    def disconnect_pin(self, input_pin, force_disconnect):
        parent_graph = self.parent_node.get_parent_graph()
        if parent_graph is None:
            return False

        if input_pin in self.connected_to:
            if force_disconnect or parent_graph.before_pins_disconnected(self, input_pin):
                self.connected_to.remove(input_pin)
                input_pin.set_connected_to(None)

                parent_graph.on_pins_disconnected(self, input_pin)

                self.pin_disconnected.emit(input_pin)

                parent_graph.after_pins_disconnected(self, input_pin)
                return True
            else:
                return False
        else:
            return True

    def disconnect_all(self, force_disconnect):
        ccount = len(self.connected_to)
        for i in range(ccount - 1, -1, -1):
            pin = self.connected_to[i]
            if not pin.disconnect_pin(force_disconnect):
                return False
        return True

    def purge_all(self):
        super().purge_all()
        self.connected_to = None

class DaggerPinCollection(DaggerBase):
    def __init__(self, parent_node, direction, topology_system):
        super().__init__()
        self.direction = direction
        self.parent_node = parent_node
        self.topology_system = topology_system
        self.pin_collection = {}
        self.ordered_pins = []
        self.pin_removed = DaggerSignal()
        self.pin_added = DaggerSignal()

    def get_topology_system(self):
        return self.topology_system

    def get_pin(self, with_name):
        return self.pin_collection.get(with_name)

    def add_pin(self, pin, name):
        if pin is None:
            return False

        pin.set_parent(self)

        if name:
            pin.set_pin_name(name)
        elif pin.get_pin_name():
            pin.set_pin_name(pin.get_instance_id())

        if pin.get_pin_name() in self.pin_collection:
            nn = ''.join([c for c in pin.get_pin_name() if not c.isdigit()])
            cc = 0
            while True:
                an = f"{nn}{cc}"
                if an not in self.pin_collection:
                    break
                cc += 1
            pin.set_pin_name(f"{nn}{cc}")

        pin.set_parent_node(self.parent_node)

        self.pin_collection[pin.get_pin_name()] = pin
        self.ordered_pins.append(pin)

        self.pin_added.emit(pin)

        return True

    def set_pin_name(self, pin, name):
        if pin.get_pin_name() == name:
            return True

        if self.get_pin(name):
            return False

        del self.pin_collection[pin.get_pin_name()]
        self.pin_collection[name] = pin

        pin.set_pin_name(name)

        return True

    def remove_pin(self, pin):
        if pin in self.ordered_pins and not pin.is_connected():
            if self.parent_node and not self.parent_node.can_remove_pin(pin):
                return False

            del self.pin_collection[pin.get_pin_name()]
            self.ordered_pins.remove(pin)
        return False

    def get_index(self, pin):
        try:
            return self.ordered_pins.index(pin)
        except ValueError:
            return -1

    def get_parent_node(self):
        return self.parent_node

    def get_pin_direction(self):
        return self.direction

    def get_all_pins(self):
        return self.ordered_pins

    def get_all_non_connected_pins(self):
        return [pin for pin in self.ordered_pins if not pin.is_connected()]

    def get_all_connected_pins(self):
        return [pin for pin in self.ordered_pins if pin.is_connected()]

    def purge_all(self):
        for pin in self.ordered_pins:
            pin.purge_all()
        self.pin_collection = None
        self.ordered_pins = None
        super().purge_all()

    def get_first_unconnected_pin(self):
        for pin in self.ordered_pins:
            if not pin.is_connected():
                return pin
        return None

def contains_pin(pins, pin):
    return pin in pins

def remove_pin(pins, pin):
    if pin in pins:
        pins.remove(pin)

class DaggerNode(DaggerBase):
    def __init__(self):
        super().__init__()
        self.current_t_system_eval = -1
        self.name = "DaggerNode"
        self.parent_graph = None
        self.descendents = [[] for _ in range(MaxTopologyCount)]
        self.subgraph_affiliation = [-1] * MaxTopologyCount
        self.ordinal = [-1] * MaxTopologyCount
        self.output_pins = [DaggerPinCollection(self, PinDirection.Output, i) for i in range(MaxTopologyCount)]
        self.input_pins = [DaggerPinCollection(self, PinDirection.Input, i) for i in range(MaxTopologyCount)]
        self.after_added_to_graph = DaggerSignal()
        self.before_added_to_graph = DaggerSignal()
        self.added_to_graph = DaggerSignal()
        self.pin_cloned = DaggerSignal()
        self.name_changed = DaggerSignal()

    def before_added_to_graph(self):
        return self.before_added_to_graph

    def after_added_to_graph(self):
        return self.after_added_to_graph

    def added_to_graph(self):
        return self.added_to_graph

    def get_parent_graph(self):
        return self.parent_graph

    def set_parent_graph(self, graph):
        self.parent_graph = graph

    def get_first_unconnected_input_pin(self, topology_system):
        return self.input_pins[topology_system].get_first_unconnected_pin()

    def get_first_unconnected_output_pin(self, topology_system):
        return self.output_pins[topology_system].get_first_unconnected_pin()

    def get_ordinal(self, topology_system):
        return self.ordinal[topology_system]

    def set_ordinal(self, topology_system, ord):
        self.ordinal[topology_system] = ord

    def get_subgraph_affiliation(self, topology_system):
        return self.subgraph_affiliation[topology_system]

    def set_subgraph_affiliation(self, topology_system, index):
        self.subgraph_affiliation[topology_system] = index

    def get_name(self):
        return self.name

    def set_name(self, new_name):
        self.name = new_name
        self.name_changed.emit(new_name)

    def get_input_pins(self, topology_system):
        return self.input_pins[topology_system]

    def get_output_pins(self, topology_system):
        return self.output_pins[topology_system]

    def get_descendents(self, topology_system):
        return self.descendents[topology_system]

    def set_descendents(self, topology_system, desc):
        self.descendents[topology_system] = desc

    def get_ascendents(self, topology_system):
        retv = []
        if self.parent_graph:
            all_nodes = self.parent_graph.get_nodes()
            for node in all_nodes:
                if node != self and self in node.get_descendents(topology_system):
                    retv.append(node)
        return retv

    def is_top_level(self, topology_system):
        all_pins = self.input_pins[topology_system].get_all_pins()
        for pin in all_pins:
            if pin.is_connected():
                if pin.get_connected_to().get_parent_node() is not None:
                    return False
        return True

    def is_bottom_level(self, topology_system):
        all_pins = self.output_pins[topology_system].get_all_pins()
        for pin in all_pins:
            if pin.is_connected():
                return False
        return True

    def disconnect_all_pins(self):
        for i in range(self.parent_graph.get_topology_count()):
            all_output = self.output_pins[i].get_all_pins()
            for j in range(len(all_output) - 1, -1, -1):
                opin = all_output[j]
                if not opin.disconnect_all(False):
                    return False

            all_input = self.input_pins[i].get_all_pins()
            for j in range(len(all_input) - 1, -1, -1):
                ipin = all_input[j]
                if not ipin.disconnect_pin(False):
                    return False

        return True

    def get_dagger_output_pin(self, with_name, topology_system):
        opin = self.output_pins[topology_system].get_pin(with_name)
        if opin:
            return opin
        return None

    def get_dagger_input_pin(self, with_name, topology_system):
        return self.input_pins[topology_system].get_pin(with_name)

    def is_true_source(self, topology_system):
        return len(self.input_pins[topology_system].get_all_pins()) == 0

    def is_true_dest(self, topology_system):
        return len(self.output_pins[topology_system].get_all_pins()) == 0

    def get_current_t_system_eval(self):
        return self.current_t_system_eval

    def set_current_t_system_eval(self, system):
        self.current_t_system_eval = system

    def should_clone_pin(self, pin):
        if pin.get_auto_clone_master():
            if pin.is_input_pin():
                to_max = pin.get_auto_clone_master().get_max_auto_clone()
                if to_max != 0:
                    if to_max == -1 or pin.get_auto_clone_master().get_auto_clone_count() < to_max:
                        return True
            else:
                opin = pin
                if len(opin.get_connected_to()) == 1:
                    to_max = pin.get_auto_clone_master().get_max_auto_clone()
                    if to_max != 0:
                        if to_max == -1 or pin.get_auto_clone_master().get_auto_clone_count() < to_max:
                            return True
        return False

    def force_clone_with_name(self, pin, pin_name):
        retv = self.clone_pin(pin, None)
        if retv:
            parent_collection = pin.get_parent()
            if not parent_collection.set_pin_name(retv, pin_name):
                self.remove_clone_pin(pin)
                retv = None
        return retv

    def rename_pin(self, pin, pin_name):
        if not pin.can_rename():
            return False

        parent_collection = pin.get_parent()
        return parent_collection.set_pin_name(pin, pin_name)

    def clone_pin(self, pin, force_auto_clone_master):
        if pin.is_input_pin():
            input_pin = force_auto_clone_master or pin.get_auto_clone_master()
            if input_pin is None:
                return None

            cloned_input = input_pin.clone()
            if cloned_input:
                cloned_input.cloned(input_pin)
                parent_collection = pin.get_parent()
                if parent_collection.add_pin(cloned_input, ""):
                    self.pin_cloned.emit(cloned_input)
                    cloned_input.on_cloned()
                    return cloned_input
        else:
            output_pin = force_auto_clone_master or pin.get_auto_clone_master()
            if output_pin is None:
                return None

            cloned_output = output_pin.clone()
            if cloned_output:
                cloned_output.cloned(output_pin)
                parent_collection = pin.get_parent()
                if parent_collection.add_pin(cloned_output, ""):
                    self.pin_cloned.emit(cloned_output)
                    cloned_output.on_cloned()
                    return cloned_output

        return None

    def should_remove_clone_pin(self, pin):
        if pin.get_auto_clone_master():
            return not pin.is_connected()
        return False

    def remove_clone_pin(self, pin):
        parent_collection = pin.get_parent()
        if pin.get_auto_clone_master() != pin:
            return parent_collection.remove_pin(pin)
        else:
            all_pins = parent_collection.get_all_non_connected_pins()
            for tpin in all_pins:
                if tpin != pin and tpin.get_auto_clone_master() == pin.get_auto_clone_master():
                    return parent_collection.remove_pin(tpin)
        return False

    def can_remove_pin(self, pin):
        return True

    def is_descendent_of(self, node, topology_system):
        return node in self.descendents[topology_system]

    def purge_all(self):
        super().purge_all()
        for i in range(MaxTopologyCount):
            if self.input_pins[i]:
                self.input_pins[i].purge_all()
            if self.output_pins[i]:
                self.output_pins[i].purge_all()
            self.descendents[i] = None
        self.output_pins = None
        self.input_pins = None
        self.descendents = None

class DaggerGraph(DaggerBase):
    def __init__(self, topology_count):
        super().__init__()
        self.nodes = []
        self.sub_graph_count = [0] * MaxTopologyCount
        self.max_ordinal = [0] * MaxTopologyCount
        self.topology_count = topology_count or 1
        self.pins_disconnected = DaggerSignal()
        self.pins_connected = DaggerSignal()
        self.node_removed = DaggerSignal()
        self.node_added = DaggerSignal()
        self.topology_changed = DaggerSignal()
        self.topology_enabled = True

        self.calculate_topology()

    def calculate_topology(self):
        self.calculate_topology_depth_first_search()

    def get_top_level_nodes(self, topology_system):
        return [node for node in self.nodes if node.is_top_level(topology_system)]

    def get_enable_topology(self):
        return self.topology_enabled

    def set_enable_topology(self, enabled):
        if enabled == self.topology_enabled:
            return
        self.topology_enabled = enabled
        self.calculate_topology()

    def get_nodes(self):
        return self.nodes

    def get_max_ordinal(self, topology_system):
        return self.max_ordinal[topology_system]

    def get_sub_graph_count(self, topology_system):
        return self.sub_graph_count[topology_system]

    def get_topology_count(self):
        return self.topology_count

    def before_pins_connected(self, connect_from, connect_to):
        return True

    def after_pins_connected(self, connect_from, connect_to):
        if connect_from.get_parent_node().should_clone_pin(connect_from):
            if connect_from.get_parent_node().clone_pin(connect_from, None) is None:
                self.emit_error("failed to autoclone pin")

        if connect_to.get_parent_node().should_clone_pin(connect_to):
            if connect_to.get_parent_node().clone_pin(connect_to, None) is None:
                self.emit_error("failed to autoclone pin")

    def before_pins_disconnected(self, connect_from, connect_to):
        return True

    def after_pins_disconnected(self, connect_from, connect_to):
        if connect_from.get_parent_node().should_remove_clone_pin(connect_from):
            if not connect_from.get_parent_node().remove_clone_pin(connect_from):
                self.emit_error("failed to remove autocloned pin")

        if connect_to.get_parent_node().should_remove_clone_pin(connect_to):
            if not connect_to.get_parent_node().remove_clone_pin(connect_to):
                self.emit_error("failed to remove autocloned pin")

    def on_pins_disconnected(self, disconnect_output, disconnect_input):
        self.calculate_topology()
        self.pins_disconnected.emit(disconnect_output.get_instance_id(), disconnect_input.get_instance_id())

    def on_pins_connected(self, connect_from, connect_to):
        self.calculate_topology()
        self.pins_connected.emit(connect_from, connect_to)

    def get_bottom_level_nodes(self, topology_system):
        return [node for node in self.nodes if node.is_bottom_level(topology_system)]

    def get_sub_graph_nodes(self, topology_system, index):
        if index > self.sub_graph_count[topology_system] - 1:
            return []
        return [node for node in self.nodes if node.get_subgraph_affiliation(topology_system) == index]

    def get_sub_graphs(self, topology_system):
        return [self.get_sub_graph_nodes(topology_system, i) for i in range(self.sub_graph_count[topology_system])]

    def get_nodes_with_ordinal(self, topology_system, ordinal):
        return [node for node in self.nodes if node.get_ordinal(topology_system) == ordinal]

    def get_nodes_with_name(self, name):
        return [node for node in self.nodes if node.get_name() == name]

    def get_pin_with_instance_id(self, pin_instance_id):
        for node in self.nodes:
            for i in range(self.topology_count):
                retv = node.get_input_pins(i).get_pin(pin_instance_id)
                if retv:
                    return retv
                retv = node.get_output_pins(i).get_pin(pin_instance_id)
                if retv:
                    return retv
        return None

    def get_node_with_instance_id(self, node_instance_id):
        for node in self.nodes:
            if node.get_instance_id() == node_instance_id:
                return node
        return None

    def all_connections(self, topology_system):
        retv = []
        for node in self.nodes:
            pins = node.get_input_pins(topology_system).get_all_pins()
            for pin in pins:
                if pin.is_connected():
                    retv.append(pin)
        return retv

    def remove_node(self, node):
        if node is None:
            return False

        if self.before_node_removed(node):
            if not node.disconnect_all_pins():
                return False

            self.nodes.remove(node)

            node.purge_all()

            self.node_removed.emit(node.get_instance_id())

            node.set_parent_graph(None)

            self.calculate_topology()

            return True
        return False

    def add_node(self, node, calculate=True):
        if node.get_parent_graph():
            return None

        node.before_added_to_graph.emit()
        node.set_parent_graph(self)
        self.nodes.append(node)

        if calculate:
            self.calculate_topology()
        else:
            for t in range(self.topology_count):
                self.sub_graph_count[t] += 1
                self.max_ordinal[t] = max(1, self.max_ordinal[t])
                node.set_subgraph_affiliation(t, self.sub_graph_count[t] + 1)
                node.set_ordinal(t, 0)
                node.set_descendents(t, [])

        self.node_added.emit(node)
        node.added_to_graph.emit()
        node.after_added_to_graph.emit()
        return node

    def add_nodes(self, nodes):
        for node in nodes:
            if node.get_parent_graph():
                return None

        for node in nodes:
            node.before_added_to_graph.emit()
            node.set_parent_graph(self)
            self.nodes.append(node)
            self.node_added.emit(node)
            node.added_to_graph.emit()
            node.after_added_to_graph.emit()
        self.calculate_topology()

        return nodes

    def before_node_removed(self, node):
        return True

    def graph_topology_changed(self):
        pass

    def calculate_topology_depth_first_search(self):
        if not self.topology_enabled:
            return

        for t in range(self.topology_count):
            self.max_ordinal[t] = 0

            for node in self.nodes:
                node.set_ordinal(t, -1)
                node.set_subgraph_affiliation(t, -1)
                node.set_descendents(t, [])

            tnodes = self.get_top_level_nodes(t)

            touched_set_list = []

            for i, node in enumerate(tnodes):
                node.set_ordinal(t, 0)

                touched_set = set()

                all_out_pins = node.get_output_pins(t).get_all_pins()
                for output in all_out_pins:
                    opin = output
                    connected_to_pins = opin.get_connected_to()

                    for inpin in connected_to_pins:
                        newset = self.recurse_calculate_topology_depth_first_search(1, inpin.get_parent_node(), touched_set, t)

                        for setnode in newset:
                            if setnode not in node.get_descendents(t):
                                node.set_descendents(t, node.get_descendents(t) + [setnode])

                        node.set_descendents(t, sorted(node.get_descendents(t), key=cmp_to_key(lambda a, b: a.get_ordinal(t) - b.get_ordinal(t))))

                touched_set.add(node)

                if i == 0:
                    touched_set_list.append(touched_set)
                else:
                    merged = False
                    for u, tset in enumerate(touched_set_list):
                        intersection = touched_set.intersection(tset)

                        if intersection:
                            touched_set_list[u] = touched_set_list[u].union(touched_set)
                            merged = True
                            break

                    if not merged:
                        touched_set_list.append(touched_set)

            for i, tset in enumerate(touched_set_list):
                for node in tset:
                    node.set_subgraph_affiliation(t, i)

            self.sub_graph_count[t] = len(touched_set_list)

        self.graph_topology_changed()
        self.topology_changed.emit()

    def recurse_calculate_topology_depth_first_search(self, level, node, touched_set, topology_system):
        retv = set()
        if node is None:
            return retv

        node.set_ordinal(topology_system, max(level, node.get_ordinal(topology_system)))
        self.max_ordinal[topology_system] = max(self.max_ordinal[topology_system], node.get_ordinal(topology_system))

        all_out_pins = node.get_output_pins(topology_system).get_all_pins()
        for output in all_out_pins:
            opin = output
            ipins = opin.get_connected_to()
            for p2 in ipins:
                newset = self.recurse_calculate_topology_depth_first_search(level + 1, p2.get_parent_node(), touched_set, topology_system)
                retv = retv.union(newset)

        touched_set.add(node)

        rslice = list(retv)
        for snode in rslice:
            if snode not in node.get_descendents(topology_system):
                node.set_descendents(topology_system, node.get_descendents(topology_system) + [snode])

        node.set_descendents(topology_system, sorted(node.get_descendents(topology_system), key=cmp_to_key(lambda a, b: a.get_ordinal(topology_system) - b.get_ordinal(topology_system))))

        retv.add(node)

        return retv

MaxTopologyCount = 2
