# Accounts Directory

This directory contains XML account files for the bot. Each XML file represents a single account with device credentials.

## XML File Format

### Android SharedPreferences Format

This is the standard Android format used by the app:

```xml
<?xml version='1.0' encoding='utf-8' standalone='yes' ?>
<map>
    <string name="deviceAccount">your_device_account_here</string>
    <string name="devicePassword">your_device_password_here</string>
</map>
```

## File Naming

- Files must have the `.xml` extension
- Filename will be displayed in the GUI
- Use descriptive names (e.g., `account1.xml`, `17P_20251102074345_3(B).xml`, etc.)

## Managing Accounts

You can manage accounts in two ways:

### 1. Through the GUI (Recommended)

- Open the bot GUI application
- Navigate to the "Accounts" tab
- Each account is displayed as a card showing:
  - Filename
  - Device Account
  - Device Password
- Use the buttons to:
  - **Add Account**: Create a new account XML file
  - **Edit**: Modify an existing account (click Edit on any card)
  - **Delete**: Remove an account file (click Delete on any card)
  - **Refresh Accounts**: Reload the account list from disk

### 2. Manual File Management

You can also manually place XML files in this directory:

1. Create a new XML file with one of the formats shown above
2. Save it in this directory
3. Click "Refresh Accounts" in the GUI to load it

## Notes

- The accounts directory is located in the same folder as the executable
- All XML files in this directory will be automatically discovered
- Invalid XML files will be skipped with a warning in the console
- Files without the `.xml` extension will be ignored
- When creating new accounts through the GUI, they will be saved in the Android SharedPreferences format

## Example Files

This directory includes a sample account file showing the correct format.