## ADDED Requirements

### Requirement: An operator can create an account from the host

An operator with shell access to the running instance SHALL be able to create an account by running a console command that takes the account's email address, its password and its role, administrator or member, and optionally a display name. When no display name is given, the account SHALL be named after the part of its address before the `@`.

The created account SHALL be active and hold the given password at once: it SHALL sign in with that address and password without completing any registration or invitation step, and the console password reset SHALL change its password the way it does for any other account.

The command SHALL create the account whether or not the instance allows open registration.

The command SHALL refuse, creating nothing, when the address is not a syntactically valid email address, when the password is shorter than eight characters, or when the role is missing or is neither administrator nor member.

Every account the command creates SHALL be recorded in the audit log with its address and role, and the entry SHALL say that the account was created from the host console.

#### Scenario: Provisioning an administrator on an instance that already holds accounts

- **WHEN** an operator creates an account with the administrator role from the console on an instance that already holds accounts
- **THEN** the account holds the administrator role and is not the bootstrap administrator
- **AND** it signs in with the given address and password
- **AND** the audit log holds an entry naming its address and role and saying it was created from the host console

#### Scenario: Provisioning while open registration is off

- **WHEN** open registration is switched off and an operator creates a member account from the console
- **THEN** the account is created and signs in

#### Scenario: Rotating the password of a provisioned account

- **WHEN** an operator resets the password of an account created from the console, using the console password reset
- **THEN** the account signs in with the new password and is refused with the old one

#### Scenario: Omitting the display name

- **WHEN** an operator creates the account `ops@example.com` without a display name
- **THEN** the account's display name is `ops`

#### Scenario: Refusing invalid input

- **WHEN** an operator runs the command with a value that is not an email address, a password of seven characters, no role, or a role other than administrator or member such as `owner`
- **THEN** the command fails, says what is wrong, and no account is created

### Requirement: An account created from the host claims an empty instance under the first-account rule

An account created from the console on an instance that holds no account SHALL be that instance's bootstrap administrator, exactly as a first registration is, and the audit log SHALL hold the same entry naming it as the instance's administrator.

On an instance that holds no account, the command SHALL refuse to create a member account and SHALL say that the first account must be an administrator. A member as the first account would leave the instance with nobody able to grant the administrator role.

When the command and a registration create the first account at the same moment, exactly one account SHALL become the bootstrap administrator. If the console-created account loses, it SHALL be created as an ordinary account with the role the operator asked for.

#### Scenario: Provisioning the first administrator of a fresh instance

- **WHEN** an operator creates an administrator account from the console on an instance that holds no account
- **THEN** that account is the recorded bootstrap administrator
- **AND** the administrator listing marks it as the bootstrap administrator
- **AND** a person who registers afterwards gets the member role

#### Scenario: The console and a registration claim an empty instance at the same moment

- **WHEN** an operator creates an administrator account from the console while a registration claims the same empty instance
- **THEN** exactly one account is the recorded bootstrap administrator
- **AND** the console-created account exists and holds the administrator role

#### Scenario: Refusing a member as the first account

- **WHEN** an operator creates a member account from the console on an instance that holds no account
- **THEN** the command fails, says the first account must be an administrator, and the instance still holds no account

### Requirement: Creating the same account from the host again changes nothing

Running the command for an address that already belongs to an active account holding the requested role and a password SHALL succeed. It SHALL report that the account already exists, that nothing changed and that its password was not changed, and SHALL leave that account's password, name and sessions untouched. A deployment can therefore run the command on every rollout.

Running the command for an address that belongs to any other account SHALL fail and change nothing. This covers an account holding the other role, a deactivated account, an invited account that has not completed sign-up, and an account without a password, such as one created through Google or GitHub.

#### Scenario: Re-running provisioning on every deployment

- **WHEN** an operator runs the same create command twice, with a different password the second time
- **THEN** the second run succeeds and reports that nothing changed and that the password was not changed
- **AND** the account still signs in with the first password

#### Scenario: The address belongs to an account with another role

- **WHEN** an operator creates an administrator account for an address that a member account already holds
- **THEN** the command fails, and that account is still a member with its old password

#### Scenario: The address belongs to a deactivated account

- **WHEN** an operator creates an account for an address whose account is deactivated
- **THEN** the command fails, and the account stays deactivated

#### Scenario: The address belongs to an invited account

- **WHEN** an operator creates an account for an address that was invited and has not completed sign-up
- **THEN** the command fails, and the account stays invited and without a password

#### Scenario: The address belongs to an account without a password

- **WHEN** an operator creates an administrator account for an address whose active administrator account signs in through Google or GitHub and has no password
- **THEN** the command fails, and the account still has no password

### Requirement: Creating an account from the host stays at host trust level and prints no secret

Creating an account without registration SHALL be available only to a caller who can already execute commands inside the running instance, the same trust level the console password reset requires. The instance SHALL NOT expose it over the network.

The command's output and log lines SHALL NOT contain the password or its hash.

#### Scenario: Reading the command's output

- **WHEN** an operator creates an account from the console, runs the same command again, and runs it once more with input it refuses
- **THEN** none of the three outputs and no log line contains the password or a password hash

### Requirement: The console password reset refuses short passwords

The console password reset SHALL refuse a password shorter than eight characters and leave the stored password unchanged, matching the minimum every other way of setting a password enforces. It SHALL keep accepting any stored address, including the placeholder address `admin` that an upgraded instance can still carry.

#### Scenario: Resetting to a short password

- **WHEN** an operator resets an account's password from the console to a value of seven characters
- **THEN** the command fails, and the account still signs in with its old password

#### Scenario: Resetting the placeholder account

- **WHEN** an operator resets the password of the account whose address is `admin` to a password of at least eight characters
- **THEN** the account signs in with the address `admin` and the new password

## MODIFIED Requirements

### Requirement: The bootstrap administrator keeps its powers over other administrators after changing address

Granting or revoking the administrator role, deactivating an account that holds it, and reactivating one SHALL remain reserved for the bootstrap administrator, and SHALL be permitted on the basis of the instance's record of which account that is. Changing the bootstrap administrator's email address SHALL NOT remove any of these powers, and SHALL NOT grant them to any other account.

The one exception is creating a new account with the administrator role from the host console, which SHALL be permitted to a caller who can execute commands inside the running instance without the bootstrap administrator's involvement. It sits at the trust level of the console password reset, which can already take over the bootstrap administrator's own account. The exception covers only new accounts: the console SHALL NOT grant or revoke the role of an existing account, and SHALL NOT deactivate or reactivate one.

#### Scenario: Appointing an administrator after the address has changed

- **WHEN** the bootstrap administrator, having changed its address, grants the administrator role to another user
- **THEN** the role is granted

#### Scenario: Another administrator still cannot manage administrators

- **WHEN** an account that holds the administrator role but is not the bootstrap administrator tries to grant the role, revoke it, or deactivate or reactivate an account holding it
- **THEN** every one of those attempts is refused, whatever address the bootstrap administrator currently holds

#### Scenario: The bootstrap administrator cannot be stripped of its role

- **WHEN** any account attempts to revoke the administrator role from the bootstrap administrator or to deactivate it
- **THEN** the attempt is refused, so the instance is never left without a recognized bootstrap administrator

#### Scenario: The console cannot promote an existing account

- **WHEN** an operator runs the console create command with the administrator role for an address that a member account already holds
- **THEN** the command fails, and that account is still a member

### Requirement: Setting up and recovering an instance are documented

The product documentation SHALL describe how an instance gets its administrator - the first account created on it - and SHALL describe how to create an account from the host console, how to reset an administrator password and how to recover a forgotten administrator address, in the same place and in every language the documentation ships. The description of account creation SHALL say that the first account created from the console on an empty instance becomes its administrator, that a repeated run with the same address and role changes nothing, and that the password is then rotated with the console reset. The public documentation SHALL also answer the question of what to do when the administrator address or password is lost, and link to those steps.

Documented command examples SHALL NOT present `admin` as the address an owner is expected to use, and the documentation SHALL NOT describe the instance as holding a user named `admin`. Where the placeholder address is mentioned at all, it SHALL be as the account an instance created before this change still carries until its owner replaces it.

#### Scenario: An owner searches the documentation

- **WHEN** an owner who cannot sign in reads the published password documentation
- **THEN** it explains both resetting the password and listing the administrator addresses
- **AND** the same guidance is reachable from the questions answered on the home page

#### Scenario: An operator automates the first administrator

- **WHEN** an operator who deploys instances from scripts reads the published password documentation in any of the six published languages
- **THEN** it shows the command that creates an account from the console with its parameters
- **AND** it says the command changes nothing when run again for the same account, and that the password is rotated with the reset command

#### Scenario: Reading about roles

- **WHEN** a reader consults the access-management documentation in any of the six published languages
- **THEN** it describes the first account created on the instance as its administrator, without claiming that account's address is `admin`
