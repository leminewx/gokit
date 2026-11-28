package command

import (
	"errors"
	"testing"
)

// 真实设备样本
const (
	// GNU coreutils（192.0.2.2，df -P -k）
	_DF_GNU_SAMPLE = `Filesystem                    1024-blocks        Used   Available Capacity Mounted on
tmpfs                            26353380        8516    26344864       1% /run
/dev/sdb2                       459329648   402289900    33633668      93% /
//192.0.2.15/data    8066259712   659819904  7406439808       9% /data
`

	// BusyBox 1.24（198.51.100.31，df -k）：超长文件系统名会把记录折成两行
	_DF_BUSYBOX_WRAPPED_SAMPLE = `Filesystem           1K-blocks      Used Available Use% Mounted on
/dev/root               253920    136124    112556  55% /
none                    153380         8    153372   0% /dev
/dev/block/platform/comip-mmc.1/by-name/userlog
                            245671     23347    205121  10% /var/romlog
/dev/block/platform/comip-mmc.1/by-name/system
                            380888    103464    269560  28% /system
`

	// BusyBox 1.24（198.51.100.31，df -P -k）：POSIX 输出不折行
	_DF_BUSYBOX_POSIX_SAMPLE = `Filesystem           1024-blocks    Used Available Capacity Mounted on
/dev/root               253920    136124    112556  55% /
/dev/block/platform/comip-mmc.1/by-name/userlog                               245671     23347    205121  10% /var/romlog
`

	// 挂载点包含空格（外接存储常见）
	_DF_MOUNT_SPACE_SAMPLE = `Filesystem     1024-blocks    Used Available Capacity Mounted on
/dev/sdb1          1024000  512000    512000      50% /mnt/My Disk
`
)

func TestParseDf(t *testing.T) {
	t.Parallel()

	t.Run("GNU 单行记录", func(t *testing.T) {
		t.Parallel()

		entries, err := ParseDf(_DF_GNU_SAMPLE)
		if err != nil {
			t.Fatalf("ParseDf(GNU) 错误 = %v", err)
		}
		if len(entries) != 3 {
			t.Fatalf("ParseDf(GNU) 条目数 = %d, want 3", len(entries))
		}

		root := entries[1]
		if root.Filesystem != "/dev/sdb2" || root.MountedOn != "/" {
			t.Errorf("根分区 = %q on %q", root.Filesystem, root.MountedOn)
		}
		if root.TotalBytes != 459329648<<10 || root.UsedBytes != 402289900<<10 || root.FreeBytes != 33633668<<10 {
			t.Errorf("根分区容量 = %+v", root)
		}
		if root.UsedPercent != 93 {
			t.Errorf("根分区使用率 = %.1f, want 93", root.UsedPercent)
		}

		// 文件系统名带 // 前缀（CIFS）时不能被截断
		if entries[2].Filesystem != "//192.0.2.15/data" {
			t.Errorf("CIFS 文件系统 = %q, want //192.0.2.15/data", entries[2].Filesystem)
		}
	})

	t.Run("BusyBox 折行记录", func(t *testing.T) {
		t.Parallel()

		entries, err := ParseDf(_DF_BUSYBOX_WRAPPED_SAMPLE)
		if err != nil {
			t.Fatalf("ParseDf(BusyBox 折行) 错误 = %v", err)
		}
		if len(entries) != 4 {
			t.Fatalf("ParseDf(BusyBox 折行) 条目数 = %d, want 4（折行记录应被合并）", len(entries))
		}

		romlog := entries[2]
		if romlog.Filesystem != "/dev/block/platform/comip-mmc.1/by-name/userlog" {
			t.Errorf("折行记录文件系统 = %q", romlog.Filesystem)
		}
		if romlog.MountedOn != "/var/romlog" || romlog.UsedPercent != 10 {
			t.Errorf("折行记录 = %+v, want /var/romlog 10%%", romlog)
		}
		if romlog.TotalBytes != 245671<<10 || romlog.UsedBytes != 23347<<10 || romlog.FreeBytes != 205121<<10 {
			t.Errorf("折行记录容量 = %+v", romlog)
		}

		if entries[3].MountedOn != "/system" || entries[3].Filesystem != "/dev/block/platform/comip-mmc.1/by-name/system" {
			t.Errorf("第二条折行记录 = %+v", entries[3])
		}
	})

	t.Run("BusyBox POSIX 输出", func(t *testing.T) {
		t.Parallel()

		entries, err := ParseDf(_DF_BUSYBOX_POSIX_SAMPLE)
		if err != nil {
			t.Fatalf("ParseDf(BusyBox -P) 错误 = %v", err)
		}
		if len(entries) != 2 {
			t.Fatalf("ParseDf(BusyBox -P) 条目数 = %d, want 2", len(entries))
		}
		if entries[1].MountedOn != "/var/romlog" || entries[1].UsedPercent != 10 {
			t.Errorf("第二条记录 = %+v", entries[1])
		}
	})

	t.Run("挂载点包含空格", func(t *testing.T) {
		t.Parallel()

		entries, err := ParseDf(_DF_MOUNT_SPACE_SAMPLE)
		if err != nil {
			t.Fatalf("ParseDf(空格挂载点) 错误 = %v", err)
		}
		if len(entries) != 1 {
			t.Fatalf("ParseDf(空格挂载点) 条目数 = %d, want 1", len(entries))
		}

		entry := entries[0]
		if entry.Filesystem != "/dev/sdb1" || entry.MountedOn != "/mnt/My Disk" {
			t.Errorf("记录 = %q on %q, want /dev/sdb1 on /mnt/My Disk", entry.Filesystem, entry.MountedOn)
		}
		if entry.TotalBytes != 1024000<<10 || entry.UsedBytes != 512000<<10 || entry.FreeBytes != 512000<<10 {
			t.Errorf("容量 = %+v", entry)
		}
	})

	t.Run("表头缺失", func(t *testing.T) {
		t.Parallel()

		if _, err := ParseDf("some random text\n"); !errors.Is(err, ErrParse) {
			t.Errorf("ParseDf(无表头) 错误 = %v, want ErrParse", err)
		}
	})
}
