# governmint
TMSP Governance App

A simple voting system that enables itself to evolve over time.

An instance of the app consists of 

- entities (with a pubkey)
- groups (collections of entities with voting power)
- open and resolved proposals

Entities can invite other entities. 
Groups are formed from existing entities and can vote on proposals for that group.
An entity with voting power is known as a member.
All voting is majority-rule.
A proposal becomes a resolution when a majority vote in favour of it.

Tx types

- proposal (propose something for a group to vote on)
- vote (vote on a proposal)

Proposals can be external proposals, in which case they contain arbitrary data,
or they can be typed, in which case the data is structured and a resolution 
has implications for the application state.

Proposal types:

- new entity
- new group
- validator set change
- software update

These types will have triggers in the application to cause events to happen;
in the case of validator set changes, an event is returned over tmsp to tendermint;
for software updates, the application process is upgraded and restarted.

Voting on software upgrades allows the system to take on new features over time,
according to the needs of its users.
