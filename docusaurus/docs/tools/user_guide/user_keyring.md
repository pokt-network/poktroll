# Setting up Password Store for Cosmos Keyring

## 1. Install `pass`

```bash
# Install the password store utility
sudo apt-get install pass
```

## 2. Create GPG Key

```bash
# Generate a new GPG key pair - you'll be prompted for:
# - Kind of key: Choose RSA
# - Key size: 3072 bits is recommended
# - Key validity: Choose your preferred duration
# - Your name and email
gpg --full-generate-key
```

## 3. Find Your GPG Key ID

```bash
# List your secret keys and find your key ID
gpg --list-secret-keys --keyid-format LONG

# The output will look like:
# sec   rsa3072/B9448E560E033C02 <-- This is your key ID
#       5F79E46574CF39CDA4FB46BDB9448E560E033C02
# uid   [ultimate] Your Name <your.email@example.com>
```

## 4. Initialize Password Store

```bash
# Initialize pass with your GPG key ID
# Replace B9448E560E033C02 with your actual key ID
pass init B9448E560E033C02
```

## 5. Store Cosmos Keyring Password

```bash
# Store your password - you will be prompted to enter it
pass insert cosmos-keyring
```

## 6. Verify Password Storage

```bash
# This will display your stored password
pass cosmos-keyring
```

## 7. Test Configuration

```bash
# Test if poktrolld can now access the keyring without prompting
poktrolld keys list
```

Note: Make sure to keep your GPG private key secure, as it's used to decrypt your stored passwords.
