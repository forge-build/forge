### Current State


* Implemented Build Controller:
   * reconcile infrastructure (to be tested)
   * reconcile connection. (to be implemented)
   * reconcile provisioners (to be implemented)
   * reconcile image exported (to be implemented)


* Implemented Infra Provider:
    * Simple Provider (to be tested)
    * AWS Provider (to be implemented)
    * GCP 
    * Azure 
    * etc...


* Implemented Provisioners:
    * Builtin Shell Provisioner
    * Ansible Provisioner
    * etc...


### Next Steps

* Test the Build Controller reconcile infrastructure logic. []
   * Build a small Infrastructure Provider to test the Build Controller reconcile infrastructure logic.
   * Test the Build Controller reconcile infrastructure logic with the simple provider. (Ensure status.Ready is synced with the infrastructure provider status)

### To Do

* Implement the reconcile connection logic.
* Implement the reconcile provisioners logic.
* Implement the reconcile image exported logic.

