package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/hyperledger-labs/yui-relayer/config"
	"github.com/hyperledger-labs/yui-relayer/core"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

func pathsCmd(ctx *config.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "paths",
		Aliases: []string{"pth"},
		Short:   "manage path configurations",
		Long: `
A path represents the "full path" or "link" for communication between two chains. This includes the client, 
connection, and channel ids from both the source and destination chains as well as the strategy to use when relaying`,
		RunE: noCommand,
	}

	cmd.AddCommand(
		pathsListCmd(ctx),
		pathsAddCmd(ctx),
		pathsEditCmd(ctx),
	)

	return cmd
}

func pathsListCmd(ctx *config.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"l"},
		Short:   "print out configured paths",
		RunE: func(cmd *cobra.Command, args []string) error {
			jsn, _ := cmd.Flags().GetBool(flagJSON)
			yml, _ := cmd.Flags().GetBool(flagYAML)
			switch {
			case yml && jsn:
				return fmt.Errorf("can't pass both --json and --yaml, must pick one")
			case yml:
				out, err := yaml.Marshal(ctx.Config.Paths)
				if err != nil {
					return err
				}
				fmt.Println(string(out))
				return nil
			case jsn:
				out, err := json.Marshal(ctx.Config.Paths)
				if err != nil {
					return err
				}
				fmt.Println(string(out))
				return nil
			default:
				// i := 0
				// for k, pth := range configInstance.Paths {
				// 	chains, err := configInstance.GetChains(pth.Src.ChainID, pth.Dst.ChainID)
				// 	if err != nil {
				// 		return err
				// 	}
				// 	stat := pth.QueryPathStatus(chains[pth.Src.ChainID], chains[pth.Dst.ChainID]).Status
				// 	printPath(i, k, pth, checkmark(stat.Chains), checkmark(stat.Clients), checkmark(stat.Connection), checkmark(stat.Channel))
				// 	i++
				// }
				// return nil
				return fmt.Errorf("not implemented error")
			}
		},
	}
	return yamlFlag(jsonFlag(cmd))
}

func pathsAddCmd(ctx *config.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "add [src-chain-id] [dst-chain-id] [path-name]",
		Aliases: []string{"a"},
		Short:   "add a path to the list of paths",
		Args:    cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			src, dst := args[0], args[1]
			_, err := ctx.Config.GetChains(src)
			if err != nil {
				return fmt.Errorf("chains need to be configured before paths to them can be added: %w", err)
			}

			file, err := cmd.Flags().GetString(flagFile)
			if err != nil {
				return err
			}

			if file != "" {
				if err := fileInputPathAdd(ctx.Config, file, args[2]); err != nil {
					return err
				}
			} else {
				if err := userInputPathAdd(ctx.Config, src, dst, args[2]); err != nil {
					return err
				}
			}

			return ctx.Config.OverWriteConfig()
		},
	}
	return fileFlag(cmd)
}

func pathsEditCmd(ctx *config.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "edit [path-name] [src or dst] [key] [value]",
		Aliases: []string{"e"},
		Short:   "Edit the config file",
		RunE: func(cmd *cobra.Command, args []string) error {
			pathName := args[0]
			srcDst := args[1]
			key := args[2]
			value := args[3]
			configPath, err := ctx.Config.Paths.Get(pathName)
			if err != nil {
				return err
			}
			var pathEnd *core.PathEnd
			switch srcDst {
			case "src":
				pathEnd = configPath.Src
			case "dst":
				pathEnd = configPath.Dst
			default:
				return fmt.Errorf("invalid src or dst: %s. Valid values are: src, dst", srcDst)
			}
			switch key {
			case "client-id":
				pathEnd.ClientID = value
			case "channel-id":
				pathEnd.ChannelID = value
			case "connection-id":
				pathEnd.ConnectionID = value
			case "port-id":
				pathEnd.PortID = value
			default:
				return fmt.Errorf("invalid key: %s. Valid keys are: client-id, channel-id, connection-id, port-id", key)
			}
			return ctx.Config.OverWriteConfig()
		},
	}
	return cmd
}

func fileInputPathAdd(config *config.Config, file, name string) error {
	// If the user passes in a file, attempt to read the chain config from that file
	p := &core.Path{}
	if _, err := os.Stat(file); err != nil {
		return err
	}

	byt, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	if err = json.Unmarshal(byt, &p); err != nil {
		return err
	}

	if err = config.Paths.Add(name, p); err != nil {
		return err
	}

	return nil
}

func userInputPathAdd(config *config.Config, src, dst, name string) error {
	var (
		value string
		err   error
		path  = &core.Path{
			Strategy: &core.StrategyCfg{Type: "naive"},
			Src: &core.PathEnd{
				ChainID: src,
				Order:   "ORDERED",
			},
			Dst: &core.PathEnd{
				ChainID: dst,
				Order:   "ORDERED",
			},
		}
	)

	fmt.Printf("enter src(%s) client-id...\n", src)
	if value, err = readStdin(); err != nil {
		return err
	}

	path.Src.ClientID = value

	if err = path.Src.Vclient(); err != nil {
		return err
	}

	fmt.Printf("enter src(%s) connection-id...\n", src)
	if value, err = readStdin(); err != nil {
		return err
	}

	path.Src.ConnectionID = value

	if err = path.Src.Vconn(); err != nil {
		return err
	}

	fmt.Printf("enter src(%s) channel-id...\n", src)
	if value, err = readStdin(); err != nil {
		return err
	}

	path.Src.ChannelID = value

	if err = path.Src.Vchan(); err != nil {
		return err
	}

	fmt.Printf("enter src(%s) port-id...\n", src)
	if value, err = readStdin(); err != nil {
		return err
	}

	path.Src.PortID = value

	if err = path.Src.Vport(); err != nil {
		return err
	}

	fmt.Printf("enter src(%s) version...\n", src)
	if value, err = readStdin(); err != nil {
		return err
	}

	path.Src.Version = value

	if err = path.Src.Vversion(); err != nil {
		return err
	}

	fmt.Printf("enter dst(%s) client-id...\n", dst)
	if value, err = readStdin(); err != nil {
		return err
	}

	path.Dst.ClientID = value

	if err = path.Dst.Vclient(); err != nil {
		return err
	}

	fmt.Printf("enter dst(%s) connection-id...\n", dst)
	if value, err = readStdin(); err != nil {
		return err
	}

	path.Dst.ConnectionID = value

	if err = path.Dst.Vconn(); err != nil {
		return err
	}

	fmt.Printf("enter dst(%s) channel-id...\n", dst)
	if value, err = readStdin(); err != nil {
		return err
	}

	path.Dst.ChannelID = value

	if err = path.Dst.Vchan(); err != nil {
		return err
	}

	fmt.Printf("enter dst(%s) port-id...\n", dst)
	if value, err = readStdin(); err != nil {
		return err
	}

	path.Dst.PortID = value

	if err = path.Dst.Vport(); err != nil {
		return err
	}

	fmt.Printf("enter dst(%s) version...\n", dst)
	if value, err = readStdin(); err != nil {
		return err
	}

	path.Dst.Version = value

	if err = path.Dst.Vversion(); err != nil {
		return err
	}

	if err = config.Paths.Add(name, path); err != nil {
		return err
	}

	return nil
}
