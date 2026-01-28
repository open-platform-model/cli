package output

import (
	"path/filepath"
	"sort"
	"strings"
)

const (
	// Tree characters
	treeEdge  = "├── "
	treeLast  = "└── "
	treeVert  = "│   "
	treeSpace = "    "

	// Description alignment column
	descriptionColumn = 30
)

// TreeNode represents a node in the file tree.
type TreeNode struct {
	Name        string
	Description string
	IsDir       bool
	Children    []*TreeNode
}

// RenderFileTree renders a file tree with descriptions aligned at column 30.
// Files is a map of relative paths to their descriptions.
// ModuleName is the root directory name.
func RenderFileTree(moduleName string, files map[string]string) string {
	if len(files) == 0 {
		return ""
	}

	// Build tree structure
	root := &TreeNode{
		Name:     moduleName,
		IsDir:    true,
		Children: []*TreeNode{},
	}

	for path, desc := range files {
		parts := strings.Split(filepath.ToSlash(path), "/")
		current := root

		for i, part := range parts {
			isLast := i == len(parts)-1

			// Find or create child
			var child *TreeNode
			for _, c := range current.Children {
				if c.Name == part {
					child = c
					break
				}
			}

			if child == nil {
				child = &TreeNode{
					Name:     part,
					IsDir:    !isLast,
					Children: []*TreeNode{},
				}
				current.Children = append(current.Children, child)
			}

			if isLast {
				child.Description = desc
			}

			current = child
		}
	}

	// Sort children alphabetically (directories first)
	sortTree(root)

	// Render tree
	var sb strings.Builder
	renderNode(&sb, root, "", true, true)
	return sb.String()
}

// sortTree recursively sorts tree nodes (directories first, then alphabetically).
func sortTree(node *TreeNode) {
	if len(node.Children) == 0 {
		return
	}

	sort.Slice(node.Children, func(i, j int) bool {
		// Directories before files
		if node.Children[i].IsDir != node.Children[j].IsDir {
			return node.Children[i].IsDir
		}
		// Alphabetical
		return node.Children[i].Name < node.Children[j].Name
	})

	for _, child := range node.Children {
		sortTree(child)
	}
}

// renderNode recursively renders a tree node with proper indentation and styling.
func renderNode(sb *strings.Builder, node *TreeNode, prefix string, isRoot, isLast bool) {
	styles := GetStyles()

	if isRoot {
		// Root node
		name := node.Name + "/"
		sb.WriteString(styles.Bold.Render(name))
		sb.WriteString("\n")
	} else {
		// Child node
		connector := treeEdge
		if isLast {
			connector = treeLast
		}

		name := node.Name
		if node.IsDir {
			name += "/"
		}

		line := prefix + connector + name

		// Add description if present, aligned to column 30
		if node.Description != "" {
			padding := descriptionColumn - len(line)
			if padding < 2 {
				padding = 2
			}
			line += strings.Repeat(" ", padding)
			line += styles.Muted.Render(node.Description)
		}

		sb.WriteString(line)
		sb.WriteString("\n")
	}

	// Render children
	for i, child := range node.Children {
		childIsLast := i == len(node.Children)-1

		var childPrefix string
		if isRoot {
			childPrefix = ""
		} else {
			if isLast {
				childPrefix = prefix + treeSpace
			} else {
				childPrefix = prefix + treeVert
			}
		}

		renderNode(sb, child, childPrefix, false, childIsLast)
	}
}

// RenderSimpleTree renders a simple tree without descriptions (for compatibility).
func RenderSimpleTree(moduleName string, files []string) string {
	fileMap := make(map[string]string)
	for _, f := range files {
		fileMap[f] = ""
	}
	return RenderFileTree(moduleName, fileMap)
}
