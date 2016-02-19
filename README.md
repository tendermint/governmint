# GovernMint

TMSP Governance Layer

A simple voting system that enables itself to evolve over time.

- *Entities* with a pubkey
- *Groups* are collections of members
- *Members* are entities associated with a group with voting power
- *Proposals* of different types:
  * *GroupUpdateProposal*: a proposal to change the group membership, etc
  * *GroupCreateProposal*: a proposal to create a new group
  * *VariableSetProposal*: a proposal to set a variable value
  * *TextProposal*: a human readible proposal
  * *SoftwareUpgradeProposal*: a proposal to upgrade software
- *Votes* are on proposals by members

Groups are formed from existing entities and can vote on proposals for that group.
An entity with voting power is known as a member of that group.

#### Tx types

- *ProposeTx* to propose something for a group to vote on
- *CastTx* to vote on a proposal
