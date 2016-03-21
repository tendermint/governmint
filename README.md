# GovernMint

TMSP Governance Layer

A simple voting system that enables itself to evolve over time.

- *Entities* are identified by a pubkey
- *Members* are entities associated with a group; can vote on proposals for that group
- *Groups* are collections of members
- *Votes* are cast on proposals by members

- *Proposal* types:
  * *GroupUpdateProposal*: change the group membership, etc
  * *GroupCreateProposal*: create a new group
  * *VariableSetProposal*: set a variable value
  * *TextProposal*: create a human readible proposal
  * *SoftwareUpgradeProposal*: upgrade software
  
#### Tx types

- *ProposeTx* to propose something for a group to vote on
- *CastTx* to vote on a proposal
