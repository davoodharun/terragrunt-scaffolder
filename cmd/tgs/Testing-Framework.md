# Testing Framework and Methodolgy


### test cases

- should be executed with the tgs generate command
- generate command creates .infrastructure folder if it does not exist
- performs validation for each stack yaml file listed in tgs.yaml
- performs validation for tgs.yaml file
- after the generate command completes successfully, there should be:
    - a .infrastructure/config folder
    - config folder has subfolders for each stack which each have  an .hcl file for each environment listed in tgs.yaml
    - a .infrastructure/_components folder
    - the _component folder has a subfolder in it for each stack listed in tgs.yaml
    - a subscription folder for each subscription listed in tgs.yaml
    - each subscription folder has a subscription.hcl file
    - region folders within each subscription folder that match the regions listed in the stack.yaml
    - each region folder has a region.hcl file
    - within each region folder there should be a folder for each environment listed in tgs.yaml (dev, test, stage etc)
    - each environment folder has an environment.hcl file in it
    - there is a sub folder in _components directory for each component listed in stack files
    - each sub folder within the _components/stackName directory has:
        - main.tf
        - providers.tf
        - variables.tf
        - component.hcl
    - within each environment folder there is an application folder listed for a component that exists in _components
    - if that application has multiple instances, there should be sub folders within that application folder; each sub folder will contain a terragrunt.hcl file with an include statement that references the respective component in the _component directory
    - if that application only has one instance, then there is a terragrunt.hcl file in that foler that references the respective component in the _component directory