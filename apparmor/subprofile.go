package apparmor

import (
	"bytes"
	"fmt"
)

// AAChangeHat transitions the current task to the specified hat.
// functional replica of aa_change_hat() in libapparmor
func AAChangeHat(hat string, token uint64) error {
	if token == 0 {
		return fmt.Errorf("invalid token(%d): must not be zero", token)
	}

	if len(hat) > 4096 {
		return fmt.Errorf("invalid hat(%s): too long", hat)
	}

	// const char *fmt = "changehat %016lx^%s";
	return SetProcAttr(gettid(), "current",
		fmt.Sprintf("changehat %016x^%s", token, hat))
}

// AAChangeProfile transitions the current task to the specified profile.
// functional replica of aa_change_profile() in libapparmor
func AAChangeProfile(profile string) error {
	// len = asprintf(&buf, "changeprofile %s", profile);
	return SetProcAttr(gettid(), "current", fmt.Sprintf("changeprofile %s", profile))
}

// AAChangeOnExec transitions the current task to the specified profile on next exec() call.
// functional replica of aa_change_onexec() in libapparmor
func AAChangeOnExec(profile string) error {
	// len = asprintf(&buf, "exec %s", profile);
	return SetProcAttr(gettid(), "exec", fmt.Sprintf("exec %s", profile))
}

// privAAChangeHatV is a replica of aa_change_hat_v() in libapparmor
// not clearly defined in documentation, thus not exported
func privAAChangeHatV(subprofiles []string, token uint64) error {
	/* setup command string which is of the form
	 * changehat <token>^hat1\0hat2\0hat3\0..\0
	 */
	var cmdBytes bytes.Buffer
	if _, err := cmdBytes.WriteString(fmt.Sprintf("changehat %016x^", token)); err != nil {
		return err
	}
	if len(subprofiles) > 0 {
		for _, subprofile := range subprofiles {
			if _, err := cmdBytes.WriteString(subprofile); err != nil {
				return err
			}

			if err := cmdBytes.WriteByte(0); err != nil {
				return err
			}
		}
	} else {
		if err := cmdBytes.WriteByte(0); err != nil {
			return err
		}
	}
	return SetProcAttr(gettid(), "current", cmdBytes.String())
}

// AAStackProfile stacks the specified profile onto the current task confinement.
// functional replica of aa_stack_profile() in libapparmor
func AAStackProfile(profile string) error {
	return SetProcAttr(gettid(), "stack", fmt.Sprintf("stack %s", profile))
}

// AAStackOnExec stacks the specified profile onto the current task confinement on next exec() call.
// functional replica of aa_get_onexec() in libapparmor
func AAStackOnExec(profile string) error {
	return SetProcAttr(gettid(), "exec", fmt.Sprintf("stack %s", profile))
}

// AAGetTaskCon returns the confinement label and mode of the specified task.
func AAGetTaskCon(pid int) (label string, mode string, err error) {
	return GetProcAttr(pid, "current")
}

// AAGetCon returns the confinement label and mode of the current task.
func AAGetCon() (label string, mode string, err error) {
	return AAGetTaskCon(gettid())
}
