# governmint

TMSP Governance Layer

A simple voting system that enables itself to evolve over time.

- Entities (with a pubkey)
- Groups (collections of members)
- Members (an entity associated with a group, with voting power)
- Proposals of different types with votes
  * GroupUpdateProposal: a proposal to change the group membership, etc
  * GroupCreateProposal: a proposal to create a new group
  * VarSetProposal: a proposal to set a variable value
  * TextProposal: a human readible proposal
  * SoftwareUpdateProposal: a proposal to update software
- Votes (votes on proposals by members)

Groups are formed from existing entities and can vote on proposals for that group.
An entity with voting power is known as a member of that group.

#### Tx types

- ProposeTx (propose something for a group to vote on)
- CastTx (vote on a proposal)
