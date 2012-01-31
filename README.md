A command-line synthesizer in Google Go.

Model
-----

Goop models a network of audio modules: Generators, which produce audio;
Effects, which modify audio; and a singleton Mixer, which sinks audio to your
speakers. Modules are connected to each other and form a directed,
probably-acyclic graph to the Mixer.

Goop is event-driven. Each autonomous module spawns a goroutine which is
responsible for processing incoming events and generating output audio. In this
way Goop avoids (in all but 1 instance) classic synchronization primitives, in
favor of channel-based communication.

Dynamic manipulation of the module graph is an important core concept. Users
are free to connect and disconnect modules in real-time, and the audio does
(should) behave as expected. In all (most?) cases, the model is a real patch
bay, and connections model real wires between components.

Implementation details
----------------------

All modules are uniquely named in the network. Events are fired to modules
according to their name.

Graph edges are modeled as "wires" carrying audio data between modules. As a
rule, audio producers "own" their output channel, and audio consumers receive a
copy of an output channel to read from. Audio channels are unbuffered, and the
only module which truly consumes audio data is the Mixer. In this way, the
network is "pull" oriented: a module blocks on its audio send channel until its
downstream module (if any exists, and is connected) requests audio data by
reading from the channel.

(Connecting one module to multiple downstream modules is not yet explicitly
dis-allowed, though it ought to be. The effect is that N goroutines are
consuming from the same channel, and each one gets on average 1/N samples. This
can have interesting, glitchy effects in the output audio.)

Disconnecting a module means closing all of its audio out channels and
re-creating them; connected downstream modules detect the close, and reset
their audio in channels (ie. to nil) accordingly.

The general network-of-modules strategy is very similar to the architecture
described in [Jim Whitehead's webpipes framework][1], except that his Chains
and Process Networks are typically defined once, at the beginning of program
execution, and not re-mapped during runtime. His Modules (called Processes) are
modeled as functions rather than structs, as they don't typically need to store
state, beyond what can be kept in the Context object passed between them.

 [1]: http://www.cs.ox.ac.uk/people/jim.whitehead/cpa2011-draft.pdf

Compatibility
-------------

goop targets weekly (eventually, will target Go 1) and may require 

    goinstall -fix

on some of its dependent libraries.

TODO
----

See the TODO file for something relatively up-to-date.
