from dagger import DaggerNode, DaggerGraph, DaggerInputPin, DaggerOutputPin, DaggerSignal, PinDirection

class CustomDaggerNode(DaggerNode):
    def __init__(self):
        super().__init__()
        self.input_pin = DaggerInputPin()
        self.input_pin.set_auto_clone(-1, "input_clone_%")
        self.output_pin1 = DaggerOutputPin()
        self.output_pin2 = DaggerOutputPin()

        self.get_input_pins(0).add_pin(self.input_pin, "input_pin")
        self.get_output_pins(0).add_pin(self.output_pin1, "output_pin1")
        self.get_output_pins(0).add_pin(self.output_pin2, "output_pin2")

def build_test_dagger_graph():
    graph = DaggerGraph(1)

    nodes = [CustomDaggerNode() for _ in range(7)]
    graph.add_nodes(nodes)

    # Connect nodes in the specified manner:
    # Node 0 connects to Node 1 and Node 2
    nodes[0].output_pin1.connect_to_input(nodes[1].input_pin)
    nodes[0].output_pin2.connect_to_input(nodes[2].input_pin)

    # Node 1 connects to Node 3 and Node 4
    nodes[1].output_pin1.connect_to_input(nodes[3].input_pin)
    nodes[1].output_pin2.connect_to_input(nodes[4].input_pin)

    # Node 2 connects to Node 5 and Node 6
    nodes[2].output_pin1.connect_to_input(nodes[5].input_pin)
    nodes[2].output_pin2.connect_to_input(nodes[6].input_pin)

    return graph

def print_graph_info(graph):
    for node in graph.get_nodes():
        print(f"Node: {node.get_instance_id()}, Name: {node.get_name()}")
        for pin in node.get_input_pins(0).get_all_pins():
            print(f"  Input Pin: {pin.get_pin_name()}, Connected: {pin.is_connected()}")
        for pin in node.get_output_pins(0).get_all_pins():
            print(f"  Output Pin: {pin.get_pin_name()}, Connected to: {[p.get_instance_id() for p in pin.get_connected_to()]}")

# Build and print the graph info
if __name__ == "__main__":
    graph = build_test_dagger_graph()
    print_graph_info(graph)
